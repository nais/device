package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/internal/random"
	"github.com/nais/device/pkg/pb"
)

func MockDevice() *pb.Device {
	return &pb.Device{
		Id:       1,
		Serial:   "mock",
		Username: "mock",
		Platform: "linux",
	}
}

type mockAuthenticator struct {
	store SessionStore
}

func (g *mockAuthenticator) Validate(_ string) error {
	return fmt.Errorf("unimplemented for mock auth")
}

func (m *mockAuthenticator) Login(ctx context.Context, _, _, _ string) (*pb.Session, error) {
	session := &pb.Session{
		Key:      random.RandomString(20, random.LettersAndNumbers),
		Expiry:   timestamppb.New(time.Now().Add(SessionDuration)),
		Groups:   []string{"group1", "group2"},
		ObjectID: "objectId123",
		Device:   MockDevice(),
	}

	err := m.store.Set(ctx, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (m *mockAuthenticator) Validator() func(http.Handler) http.Handler {
	// not used by current versions of device-agent.
	return func(handler http.Handler) http.Handler {
		return handler
	}
}

func (m *mockAuthenticator) LoginHTTP(http.ResponseWriter, *http.Request) {
	// not used by current versions of device-agent.
}

func (m *mockAuthenticator) AuthURL(http.ResponseWriter, *http.Request) {
	// not used by current versions of device-agent.
}

func NewMockAuthenticator(store SessionStore) Authenticator {
	return &mockAuthenticator{
		store: store,
	}
}
