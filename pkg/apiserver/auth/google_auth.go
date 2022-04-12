package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/auth"
	"github.com/nais/device/pkg/pb"

	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/random"
)

type googleAuthenticator struct {
	db     database.APIServer
	store  SessionStore
	google *auth.Google
}

func NewGoogleAuthenticator(google *auth.Google, db database.APIServer, store SessionStore) Authenticator {
	return &googleAuthenticator{
		db:     db,
		store:  store,
		google: google,
	}
}

func (g *googleAuthenticator) Login(ctx context.Context, token, serial, platform string) (*pb.Session, error) {
	user, err := g.google.ParseAndValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("parse and validate token: %w", err)
	}

	device, err := g.db.ReadDeviceBySerialPlatform(ctx, serial, platform)
	if err != nil {
		return nil, fmt.Errorf("read device (%s, %s), user: %s, err: %v", serial, platform, user.Email, err)
	}

	session := &pb.Session{
		Key:      random.RandomString(20, random.LettersAndNumbers),
		Expiry:   timestamppb.New(time.Now().Add(SessionDuration)),
		Groups:   user.Groups,
		ObjectID: user.ID,
		Device:   device,
	}

	err = g.store.Set(ctx, session)
	if err != nil {
		log.Errorf("Persisting session info %v: %v", session, err)
		// don't abort auth here as this might be OK
		// fixme: we must abort auth here as the database didn't accept the session, and further usage will probably fail
		return nil, fmt.Errorf("persist session: %s", err)
	}

	return session, nil
}

func (g *googleAuthenticator) Validator() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return nil
	}
}

func (g *googleAuthenticator) LoginHTTP(w http.ResponseWriter, r *http.Request) {}

func (g *googleAuthenticator) AuthURL(w http.ResponseWriter, r *http.Request) {}
