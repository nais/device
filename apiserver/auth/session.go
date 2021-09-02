package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/nais/device/apiserver/config"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/pkg/random"
)

const (
	SessionDuration     = time.Hour * 10
	HeaderKeyListenPort = "x-naisdevice-listen-port"
)

type Sessions struct {
	DB             *database.APIServerDB
	OAuthConfig    *oauth2.Config
	tokenValidator jwt.Keyfunc
	devMode        bool

	State     map[string]bool
	stateLock sync.Mutex

	Active     map[string]*database.SessionInfo
	activeLock sync.Mutex
}

func New(cfg config.Config, validator jwt.Keyfunc, db *database.APIServerDB) (*Sessions, error) {
	return &Sessions{
		DB:             db,
		devMode:        cfg.DevMode,
		tokenValidator: validator,
		State:          make(map[string]bool),
		Active:         make(map[string]*database.SessionInfo),
		OAuthConfig: &oauth2.Config{
			// RedirectURL:  "http://localhost",  don't set this
			ClientID:     cfg.Azure.ClientID,
			ClientSecret: cfg.Azure.ClientSecret,
			Scopes:       []string{"openid", fmt.Sprintf("%s/.default", cfg.Azure.ClientID)},
			Endpoint:     endpoints.AzureAD("62366534-1ec3-4962-8869-9b5535279d0b"),
		},
	}, nil
}

func (s *Sessions) Validator() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionKey := r.Header.Get("x-naisdevice-session-key")

			s.activeLock.Lock()
			defer s.activeLock.Unlock()
			sessionInfo, ok := s.Active[sessionKey]
			if !ok {
				si, err := s.DB.ReadSessionInfo(r.Context(), sessionKey)
				if err != nil {
					log.Errorf("reading session info from db: %v", err)
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				s.Active[sessionKey] = si // cache it
				sessionInfo = si
				ok = true
			}

			if !ok || !sessionInfo.Expired() {
				log.Infof("session expired: %v", sessionInfo)
				log.Infof("s: %v", s.Active)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), "sessionInfo", sessionInfo))

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Sessions) validAuthState(state string) error {
	if len(state) == 0 {
		return fmt.Errorf("no 'state' query param in auth request")
	}

	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	if _, ok := s.State[state]; ok {
		delete(s.State, state)
	} else {
		return fmt.Errorf("state not found (CSRF attack?): %v", state)
	}

	return nil
}

func parseListenPort(port string) (int, error) {
	if len(port) == 0 {
		port = "51800"
	}

	portAsNumber, err := strconv.Atoi(port)
	if err != nil {
		return -1, fmt.Errorf("parsing port '%v': %v", port, err)
	}

	return portAsNumber, err
}

func (s *Sessions) getToken(ctx context.Context, code, redirectUri string) (*oauth2.Token, error) {
	if len(code) == 0 {
		return nil, fmt.Errorf("no 'code' query param in auth request")
	}

	token, err := s.OAuthConfig.Exchange(ctx, code, oauth2.SetAuthURLParam("redirect_uri", redirectUri))
	if err != nil {
		return nil, fmt.Errorf("exchanging code for token: %w", err)
	}

	return token, nil
}

func (s *Sessions) Login(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	err := s.validAuthState(r.URL.Query().Get("state"))
	if err != nil {
		authFailed(w, "Validating auth state: %v", err)
		return
	}

	sessionInfo := &database.SessionInfo{
		Key:    random.RandomString(20, random.LettersAndNumbers),
		Expiry: time.Now().Add(SessionDuration).Unix(),
	}

	if !s.devMode {
		listenPort, err := parseListenPort(r.Header.Get(HeaderKeyListenPort))
		if err != nil {
			authFailed(w, "unable to parse listening port: %v", err)
			return
		}

		redirectUri := fmt.Sprintf("http://localhost:%d", listenPort)
		token, err := s.getToken(ctx, r.URL.Query().Get("code"), redirectUri)
		if err != nil {
			authFailed(w, "Exchanging code for token: %v", err)
			return
		}

		username, objectId, groups, err := s.parseToken(token)
		if err != nil {
			authFailed(w, "Parsing token: %v", err)
			return
		}
		sessionInfo.ObjectId = objectId

		approvalOK := false
		for _, group := range groups {
			if group == config.NaisDeviceApprovalGroup {
				approvalOK = true
			}
		}

		if !approvalOK {
			authFailed(w, "do's and don'ts not accepted, visit: https://naisdevice-approval.nais.io/ to read and accept")
			return
		}

		serial := r.Header.Get("x-naisdevice-serial")
		platform := r.Header.Get("x-naisdevice-platform")
		device, err := s.DB.ReadDeviceBySerialPlatformUsername(ctx, serial, platform, username)
		if err != nil {
			authFailed(w, "getting device: %v", err)
			return
		}

		sessionInfo.Groups = groups
		sessionInfo.Device = device
	} else {
		sessionInfo.Groups = []string{"group1", "group2"}
		sessionInfo.ObjectId = "objectId1"
		sessionInfo.Device = &database.Device{ID: 0, Serial: "serial1", Username: "username1", Platform: "platform1"}
	}

	b, err := json.Marshal(sessionInfo)
	if err != nil {
		authFailed(w, "Marshalling json: %v", err)
		return
	}

	s.activeLock.Lock()
	defer s.activeLock.Unlock()
	s.Active[sessionInfo.Key] = sessionInfo

	err = s.DB.AddSessionInfo(r.Context(), sessionInfo)
	if err != nil {
		log.Errorf("Persisting session info %v: %v", sessionInfo, err)
		// don't abort auth here as this might be OK
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		log.Errorf("writing response: %v", err)
	}

	log.Infof("login: %v", s.Active)
}

func (s *Sessions) AuthURL(w http.ResponseWriter, r *http.Request) {
	state := random.RandomString(20, random.LettersAndNumbers)
	s.stateLock.Lock()
	s.State[state] = true
	s.stateLock.Unlock()

	var authURL string
	listenPort, err := parseListenPort(r.Header.Get(HeaderKeyListenPort))
	if err != nil {
		authFailed(w, "unable to parse listening port: %s", err)
		return
	}

	if !s.devMode {
		redirectUri := fmt.Sprintf("http://localhost:%d", listenPort)
		// Override redirect_url with custom port uri
		authURL = s.OAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("redirect_uri", redirectUri))
	} else {
		authURL = fmt.Sprintf("http://localhost:%d/?state=%s&code=dev", listenPort, state)
	}

	asUrl, err := url.Parse(authURL)
	if err != nil {
		log.Errorf("parsing auth url: %v", err)
	}

	_, err = w.Write([]byte(asUrl.String()))
	if err != nil {
		log.Errorf("responding to %v %v : %v", r.Method, r.URL.Path, err)
	}
}

func (s *Sessions) parseToken(token *oauth2.Token) (string, string, []string, error) {
	var claims jwt.MapClaims
	_, err := jwt.ParseWithClaims(token.AccessToken, &claims, s.tokenValidator)
	if err != nil {
		return "", "", nil, fmt.Errorf("parsing token with claims: %v", err)
	}

	var groups []string
	groupInterface := claims["groups"].([]interface{})
	groups = make([]string, len(groupInterface))
	for i, v := range groupInterface {
		groups[i] = v.(string)
	}

	username := claims["preferred_username"].(string)
	objectId := claims["oid"].(string)

	return username, objectId, groups, nil
}

func authFailed(w http.ResponseWriter, format string, args ...interface{}) {
	w.WriteHeader(http.StatusForbidden)
	log.Warnf(format, args...)
}
