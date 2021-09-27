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

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/pb"

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
	LoginHTTP(w http.ResponseWriter, r *http.Request)
	Login(ctx context.Context, token, serial, platform string) (*pb.Session, error)
	AuthURL(w http.ResponseWriter, r *http.Request)
}

type authenticator struct {
	OAuthConfig *oauth2.Config
	db          database.APIServer
	store       SessionStore
	states      map[string]interface{}
	stateLock   sync.Mutex
	jwks        jwk.Set
	Azure       config.Azure
}

func NewAuthenticator(cfg config.Config, db database.APIServer, store SessionStore, jwks jwk.Set) Authenticator {
	return &authenticator{
		db:     db,
		store:  store,
		states: make(map[string]interface{}),
		jwks:   jwks,
		Azure:  cfg.Azure,
		OAuthConfig: &oauth2.Config{
			// RedirectURL:  "http://localhost",  don't set this
			ClientID:     cfg.Azure.ClientID,
			ClientSecret: cfg.Azure.ClientSecret,
			Scopes:       []string{"openid", fmt.Sprintf("%s/.default", cfg.Azure.ClientID)},
			Endpoint:     endpoints.AzureAD(cfg.Azure.Tenant),
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

func (s *authenticator) LoginHTTP(w http.ResponseWriter, r *http.Request) {
	err := s.validAuthState(r.URL.Query().Get("state"))
	if err != nil {
		authFailed(w, "Validating auth state: %v", err)
		return
	}

	listenPort, err := parseListenPort(r.Header.Get(HeaderKeyListenPort))
	if err != nil {
		authFailed(w, "unable to parse listening port: %v", err)
		return
	}

	redirectUri := fmt.Sprintf("http://localhost:%d", listenPort)
	token, err := s.getToken(r.Context(), r.URL.Query().Get("code"), redirectUri)
	if err != nil {
		authFailed(w, "Exchanging code for token: %v", err)
		return
	}

	serial := r.Header.Get(HeaderKeySerial)
	platform := r.Header.Get(HeaderKeyPlatform)
	session, err := s.Login(r.Context(), token.AccessToken, serial, platform)
	if err != nil {
		authFailed(w, "login: %s", err)
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(LegacySessionFromProtobuf(session))
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

func (s *authenticator) Login(ctx context.Context, token, serial, platform string) (*pb.Session, error) {
	parsedToken, err := jwt.Parse(
		[]byte(token),
		jwt.WithKeySet(s.jwks),
		jwt.WithAcceptableSkew(5*time.Second),
		jwt.WithIssuer(s.Azure.Issuer()),
		jwt.WithAudience(s.Azure.ClientID),
		jwt.WithValidate(true),
	)

	if err != nil {
		return nil, fmt.Errorf("parse token: %s", err)
	}

	claims, err := parsedToken.AsMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("convert claims to map: %s", err)
	}

	var groups []string
	approvalOK := false
	for _, group := range claims["groups"].([]interface{}) {
		s := group.(string)
		if s == config.NaisDeviceApprovalGroup {
			approvalOK = true
		}
		groups = append(groups, s)
	}

	if !approvalOK {
		return nil, fmt.Errorf("do's and don'ts not accepted, visit: https://naisdevice-approval.nais.io/ to read and accept")
	}

	username := claims["preferred_username"].(string)

	device, err := s.db.ReadDeviceBySerialPlatform(ctx, serial, platform)
	if err != nil {
		return nil, fmt.Errorf("read device (%s, %s), user: %s, err: %v", serial, platform, username, err)
	}

	session := &pb.Session{
		Key:      random.RandomString(20, random.LettersAndNumbers),
		Expiry:   timestamppb.New(time.Now().Add(SessionDuration)),
		Groups:   groups,
		ObjectID: claims["oid"].(string),
		Device:   device,
	}

	err = s.store.Set(ctx, session)
	if err != nil {
		log.Errorf("Persisting session info %v: %v", session, err)
		// don't abort auth here as this might be OK
		// fixme: we must abort auth here as the database didn't accept the session, and further usage will probably fail
		return nil, fmt.Errorf("persist session: %s", err)
	}

	return session, nil
}

func authFailed(w http.ResponseWriter, format string, args ...interface{}) {
	w.WriteHeader(http.StatusForbidden)
	log.Warnf(format, args...)
}
