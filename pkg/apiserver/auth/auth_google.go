package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/auth"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/random"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type googleAuth struct {
	db     database.APIServer
	store  SessionStore
	google *auth.Google
}

func NewGoogleAuthenticator(googleConfig *auth.Google, db database.APIServer, store SessionStore) Authenticator {
	return &googleAuth{
		db:     db,
		store:  store,
		google: googleConfig,
	}
}

func (g *googleAuth) Login(ctx context.Context, token, serial, platform string) (*pb.Session, error) {
	user, err := g.google.ParseAndValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("parse and validate token: %w", err)
	}

	device, err := g.db.ReadDeviceBySerialPlatform(ctx, serial, platform)
	if err != nil {
		return nil, fmt.Errorf("read device (%s, %s), user: %s, err: %v", serial, platform, user.Email, err)
	}

	if !strings.EqualFold(user.Email, device.Username) {
		log.Errorf("GREP: username (%s) does not match device username (%s) id (%d)", user.Email, device.Username, device.Id)
		// return nil, fmt.Errorf("username (%s) does not match device username (%s)", username, device.Username)
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
