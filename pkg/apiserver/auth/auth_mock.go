package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/random"
)

type mockAuthenticator struct {
	store SessionStore
}

func (m *mockAuthenticator) Validator() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionKey := r.Header.Get(HeaderKeySessionKey)

			sessionInfo, err := m.store.Get(r.Context(), sessionKey)
			if err != nil {
				log.Errorf("read session info: %v", err)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), "sessionInfo", sessionInfo))

			next.ServeHTTP(w, r)
		})
	}
}

func (m *mockAuthenticator) Login(w http.ResponseWriter, r *http.Request) {
	session := &pb.Session{
		Key:      random.RandomString(20, random.LettersAndNumbers),
		Expiry:   timestamppb.New(time.Now().Add(SessionDuration)),
		Groups:   []string{"group1", "group2"},
		ObjectID: "objectId123",
		Device: &pb.Device{
			Id:       1,
			Serial:   "mock",
			Username: "mock",
			Platform: "linux",
		},
	}

	err := m.store.Set(r.Context(), session)
	if err != nil {
		authFailed(w, "cache session: %v", err)
		return
	}

	err = json.NewEncoder(w).Encode(LegacySessionFromProtobuf(session))
	if err != nil {
		authFailed(w, "write response: %v", err)
		return
	}
}

func (m *mockAuthenticator) AuthURL(w http.ResponseWriter, r *http.Request) {
	port, err := parseListenPort(r.Header.Get(HeaderKeyListenPort))
	if err != nil {
		_, err = fmt.Fprintf(w, "Unable to parse header: %s", HeaderKeyListenPort)
		if err != nil {
			log.Errorf("Writing bad request response: %s", err)
		}

		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = fmt.Fprintf(w, "http://localhost:%d/?mock=true", port)
	if err != nil {
		log.Errorf("Writing auth url response: %s", err)
	}
}

func NewMockAuthenticator(store SessionStore) Authenticator {
	return &mockAuthenticator{
		store: store,
	}
}
