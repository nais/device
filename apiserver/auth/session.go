package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/nais/device/apiserver/azure/discovery"
	"github.com/nais/device/apiserver/azure/validate"
	"github.com/nais/device/apiserver/config"
	"github.com/nais/device/pkg/random"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	"net/http"
	"sync"
	"time"
)

type SessionInfo struct {
	Key      string `json:"key"`
	Expiry   int64  `json:"expiry"`
	DeviceID string `json:"deviceID"`
	Serial   string
	Platform string
	Username string
	Groups   []string
}

const SessionDuration = time.Hour * 10

type Sessions struct {
	db             *pgxpool.Pool
	oauthConfig    *oauth2.Config
	tokenValidator jwt.Keyfunc

	state     map[string]bool
	stateLock sync.Mutex

	active     map[string]SessionInfo
	activeLock sync.Mutex
}

func New(ctx context.Context, cfg config.Config) (*Sessions, error) {
	db, err := pgxpool.Connect(ctx, cfg.DbConnURI)
	if err != nil {
		return nil, fmt.Errorf("connecting to database %w", err)
	}

	tokenValidator, err := createJWTValidator(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating JWT validator: %w", err)
	}

	return &Sessions{
		db:             db,
		tokenValidator: tokenValidator,
		state:          make(map[string]bool),
		active:         make(map[string]SessionInfo),
		oauthConfig: &oauth2.Config{
			RedirectURL:  "http://localhost:51800",
			ClientID:     cfg.Azure.ClientID,
			ClientSecret: cfg.Azure.ClientSecret,
			Scopes:       []string{"openid", "6e45010d-2637-4a40-b91d-d4cbb451fb57/.default"},
			Endpoint:     endpoints.AzureAD("62366534-1ec3-4962-8869-9b5535279d0b"),
		},
	}, nil
}

func (si *SessionInfo) Expired() bool {
	return time.Unix(si.Expiry, 0).After(time.Now())
}

func (s *Sessions) Validator(ctx context.Context) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionKey := r.Header.Get("x-naisdevice-session-key")

			s.activeLock.Lock()
			defer s.activeLock.Unlock()
			sessionInfo, ok := s.active[sessionKey]
			if !ok || !sessionInfo.Expired() {
				log.Info("session expired: %v", sessionInfo)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			/*
				sessionRow := s.db.QueryRow(ctx, `SELECT key, device_id, expires FROM session WHERE key = $1`, sessionKey)

				var session SessionInfo
				err := sessionRow.Scan(&session.Key, &session.DeviceID, &session.Expiry)
				if err != nil {
					log.Infof("no session found: %v", err)
					w.WriteHeader(http.StatusUnauthorized)
					return
				}

				if session.Expiry < time.Now().Unix() {
					log.Infof("session epired: %v", session)
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
			*/

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

	if _, ok := s.state[state]; ok {
		delete(s.state, state)
	} else {
		return fmt.Errorf("state not found (CSRF attack?): %v", state)
	}

	return nil
}

func (s *Sessions) getToken(ctx context.Context, code string) (*oauth2.Token, error) {
	if len(code) == 0 {
		return nil, fmt.Errorf("no 'code' query param in auth request")
	}

	token, err := s.oauthConfig.Exchange(ctx, code)
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

	token, err := s.getToken(ctx, r.URL.Query().Get("code"))
	if err != nil {
		authFailed(w, "Exchanging code for token: %v", err)
		return
	}

	username, groups, err := s.parseToken(token)
	if err != nil {
		authFailed(w, "Parsing token: %v", err)
		return
	}

	sessionInfo := SessionInfo{
		Key:      random.RandomString(20, random.LettersAndNumbers),
		Expiry:   time.Now().Add(SessionDuration).Unix(),
		Serial:   r.Header.Get("x-naisdevice-serial"),
		Platform: r.Header.Get("x-naisdevice-platform"),
		Username: username,
		Groups:   groups,
	}

	b, err := json.Marshal(sessionInfo)
	if err != nil {
		authFailed(w, "Marshalling json: %v", err)
		return
	}

	s.activeLock.Lock()
	defer s.activeLock.Unlock()
	s.active[sessionInfo.Key] = sessionInfo

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)
	if err != nil {
		log.Errorf("writing response: %v", err)
	}
}

func (s *Sessions) StartLogin(w http.ResponseWriter, r *http.Request) {
	state := random.RandomString(20, random.LettersAndNumbers)
	s.stateLock.Lock()
	s.state[state] = true
	s.stateLock.Unlock()

	authURL := s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	_, err := w.Write([]byte(authURL))
	if err != nil {
		log.Errorf("responding to GET /login: %v", err)
	}
}

func (s *Sessions) parseToken(token *oauth2.Token) (string, []string, error) {
	var claims jwt.MapClaims
	_, err := jwt.ParseWithClaims(token.AccessToken, &claims, s.tokenValidator)
	if err != nil {
		return "", nil, fmt.Errorf("parsing token with claims: %v", err)
	}

	var groups []string
	groupInterface := claims["groups"].([]interface{})
	groups = make([]string, len(groupInterface))
	for i, v := range groupInterface {
		groups[i] = v.(string)
	}

	username := claims["preferred_username"].(string)

	return username, groups, nil
}

func createJWTValidator(conf config.Config) (jwt.Keyfunc, error) {

	if len(conf.Azure.ClientID) == 0 || len(conf.Azure.DiscoveryURL) == 0 {
		return nil, fmt.Errorf("missing required azure configuration")
	}

	certificates, err := discovery.FetchCertificates(conf.Azure)
	if err != nil {
		return nil, fmt.Errorf("retrieving azure ad certificates for token validation: %v", err)
	}

	return validate.JWTValidator(certificates, conf.Azure.ClientID), nil
}

func authFailed(w http.ResponseWriter, format string, args ...interface{}) {
	w.WriteHeader(http.StatusForbidden)
	log.Warnf(format, args...)
}
