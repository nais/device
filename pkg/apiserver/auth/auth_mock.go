package auth

import (
	"context"
	"net/http"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/random"
)

type mockAuthenticator struct {
	store SessionStore
}

func (m *mockAuthenticator) Login(ctx context.Context, token, serial, platform string) (*pb.Session, error) {
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

	err := m.store.Set(ctx, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (m *mockAuthenticator) Validator() func(http.Handler) http.Handler {
	// not used by current versions of device-agent.
	return nil
}

func (m *mockAuthenticator) LoginHTTP(w http.ResponseWriter, r *http.Request) {
	// not used by current versions of device-agent.
}

func (m *mockAuthenticator) AuthURL(w http.ResponseWriter, r *http.Request) {
	// not used by current versions of device-agent.
}

func NewMockAuthenticator(store SessionStore) Authenticator {
	return &mockAuthenticator{
		store: store,
	}
}
