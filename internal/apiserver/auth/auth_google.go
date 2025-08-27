package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/auth"
	"github.com/nais/device/internal/random"
	"github.com/nais/device/pkg/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type googleAuth struct {
	db     database.Database
	store  SessionStore
	google *auth.Google
}

func NewGoogleAuthenticator(googleConfig *auth.Google, db database.Database, store SessionStore) Authenticator {
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
		return nil, fmt.Errorf("username (%s) does not match device username (%s)", user.Email, device.Username)
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
		return nil, fmt.Errorf("persist session: %s", err)
	}

	return session, nil
}
