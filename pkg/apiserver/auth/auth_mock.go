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
	session *pb.Session
}

func (m *mockAuthenticator) Validator() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m.session == nil {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), "sessionInfo", m.session))

			next.ServeHTTP(w, r)
		})
	}
}

func (m *mockAuthenticator) Login(w http.ResponseWriter, r *http.Request) {
	//serial := w.Header().Get(HeaderKeyPlatform)
	//platform := w.Header().Get(HeaderKeySerial)
	m.session = &pb.Session{
		Key:      random.RandomString(20, random.LettersAndNumbers),
		Expiry:   timestamppb.New(time.Now().Add(SessionDuration)),
		Groups:   []string{"group1", "group2"},
		ObjectID: "objectId123",
		// fixme: mock data lives in fixture
		Device:   &pb.Device{
			Id: 1,
			Serial: "mock",
			Username: "mock",
			Platform: "linux",
		},
	}

	err := json.NewEncoder(w).Encode(LegacySessionFromProtobuf(m.session))
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

func Mock() Authenticator {
	return &mockAuthenticator{
		session: nil,
	}
}
