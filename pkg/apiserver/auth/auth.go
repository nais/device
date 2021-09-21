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

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/pb"

	"github.com/golang-jwt/jwt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/nais/device/pkg/apiserver/config"
	"github.com/nais/device/pkg/random"
)

const (
	SessionDuration     = time.Hour * 10
	HeaderKeyListenPort = "x-naisdevice-listen-port"
	HeaderKeySessionKey = "x-naisdevice-session-key"
	HeaderKeySerial     = "x-naisdevice-serial"
	HeaderKeyPlatform   = "x-naisdevice-platform"
)

type Authenticator interface {
	Validator() func(next http.Handler) http.Handler
	Login(w http.ResponseWriter, r *http.Request)
	AuthURL(w http.ResponseWriter, r *http.Request)
}

type authenticator struct {
	OAuthConfig    *oauth2.Config
	db             database.APIServer
	store          SessionStore
	tokenValidator jwt.Keyfunc
	states         map[string]interface{}
	stateLock      sync.Mutex
}

func NewAuthenticator(cfg config.Config, validator jwt.Keyfunc, db database.APIServer, store SessionStore) Authenticator {
	return &authenticator{
		db:             db,
		store:          store,
		states:         make(map[string]interface{}),
		tokenValidator: validator,
		OAuthConfig: &oauth2.Config{
			// RedirectURL:  "http://localhost",  don't set this
			ClientID:     cfg.Azure.ClientID,
			ClientSecret: cfg.Azure.ClientSecret,
			Scopes:       []string{"openid", fmt.Sprintf("%s/.default", cfg.Azure.ClientID)},
			Endpoint:     endpoints.AzureAD("62366534-1ec3-4962-8869-9b5535279d0b"),
		},
	}
}

func (s *authenticator) Validator() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionKey := r.Header.Get(HeaderKeySessionKey)

			sessionInfo, err := s.store.Get(r.Context(), sessionKey)
			if err != nil {
				log.Errorf("read session info: %v", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if sessionInfo.Expired() {
				log.Infof("session expired: %v", sessionInfo)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), "sessionInfo", sessionInfo))

			next.ServeHTTP(w, r)
		})
	}
}

// callable only once for every state param.
func (s *authenticator) validAuthState(state string) error {
	if len(state) == 0 {
		return fmt.Errorf("no 'state' query param in auth request")
	}

	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	_, ok := s.states[state]

	if !ok {
		return fmt.Errorf("state not found (CSRF attack?): %v", state)
	}

	delete(s.states, state)

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

func (s *authenticator) getToken(ctx context.Context, code, redirectUri string) (*oauth2.Token, error) {
	if len(code) == 0 {
		return nil, fmt.Errorf("no 'code' query param in auth request")
	}

	token, err := s.OAuthConfig.Exchange(ctx, code, oauth2.SetAuthURLParam("redirect_uri", redirectUri))
	if err != nil {
		return nil, fmt.Errorf("exchanging code for token: %w", err)
	}

	return token, nil
}

func (s *authenticator) Login(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	err := s.validAuthState(r.URL.Query().Get("state"))
	if err != nil {
		authFailed(w, "Validating auth state: %v", err)
		return
	}

	sessionInfo := &pb.Session{
		Key:    random.RandomString(20, random.LettersAndNumbers),
		Expiry: timestamppb.New(time.Now().Add(SessionDuration)),
	}

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
	sessionInfo.ObjectID = objectId

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

	serial := r.Header.Get(HeaderKeySerial)
	platform := r.Header.Get(HeaderKeyPlatform)
	device, err := s.db.ReadDeviceBySerialPlatformUsername(ctx, serial, platform, username)
	if err != nil {
		authFailed(w, "getting device: %v", err)
		return
	}

	sessionInfo.Groups = groups
	sessionInfo.Device = device

	err = s.store.Set(r.Context(), sessionInfo)
	if err != nil {
		log.Errorf("Persisting session info %v: %v", sessionInfo, err)
		// don't abort auth here as this might be OK
		// fixme: we must abort auth here as the database didn't accept the session, and further usage will probably fail
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(LegacySessionFromProtobuf(sessionInfo))
	if err != nil {
		log.Errorf("writing response: %v", err)
	}
}

func (s *authenticator) AuthURL(w http.ResponseWriter, r *http.Request) {
	state := random.RandomString(20, random.LettersAndNumbers)
	s.stateLock.Lock()
	s.states[state] = new(interface{})
	s.stateLock.Unlock()

	listenPort, err := parseListenPort(r.Header.Get(HeaderKeyListenPort))
	if err != nil {
		authFailed(w, "unable to parse listening port: %s", err)
		return
	}

	redirectUri := fmt.Sprintf("http://localhost:%d", listenPort)
	// Override redirect_url with custom port uri
	authURL := s.OAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("redirect_uri", redirectUri))

	asUrl, err := url.Parse(authURL)
	if err != nil {
		log.Errorf("parsing auth url: %v", err)
	}

	_, err = w.Write([]byte(asUrl.String()))
	if err != nil {
		log.Errorf("responding to %v %v : %v", r.Method, r.URL.Path, err)
	}
}

func (s *authenticator) parseToken(token *oauth2.Token) (string, string, []string, error) {
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
