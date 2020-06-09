package session

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	"io/ioutil"
	"net/http"
	"time"
)

type Session struct {
	key      string
	deviceId string
	expires  int64
}

type Sessions struct {
	db          *pgxpool.Pool
	oauthConfig *oauth2.Config
}

func New(ctx context.Context, connString, clientID, clientSecret string) (*Sessions, error) {
	db, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("connecting to database %w", err)
	}

	return &Sessions{
		db: db,
		oauthConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       []string{"openid", "6e45010d-2637-4a40-b91d-d4cbb451fb57/.default", "offline_access"},
			Endpoint:     endpoints.AzureAD("62366534-1ec3-4962-8869-9b5535279d0b"),
		},
	}, nil
}

func (s *Sessions) Validator(ctx context.Context) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionKey := r.Header.Get("x-naisdevice-session-key")

			sessionRow := s.db.QueryRow(ctx, `SELECT key, device_id, expires FROM session WHERE key = $1`, sessionKey)

			var session Session
			err := sessionRow.Scan(&session.key, &session.deviceId, &session.expires)
			if err != nil {
				log.Infof("no session found: %v", err)
				authURL := s.oauthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
				invalidSession(w, authURL)
				return
			}

			if session.expires < time.Now().Unix() {
				log.Infof("session epired: %v", session)
				authURL := s.oauthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
				invalidSession(w, authURL)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Sessions) Login(w http.ResponseWriter, r *http.Request) {
	code, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Infof("login code: %v", string(code))

	ctx := context.Background()

	token, err := s.oauthConfig.Exchange(ctx, string(code))
	if err != nil {
		log.Errorf("exchanging code for token: %v", err)
		return
	}
	log.Infof("token: %v", token)

	//randomString := random.RandomString(16, random.LettersAndNumbers)
}

func invalidSession(w http.ResponseWriter, authURL string) {
	w.Header().Add("Location", authURL)
	w.WriteHeader(http.StatusFound)
}
