package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/auth"
	"github.com/nais/device/internal/random"
	"github.com/nais/device/pkg/pb"
)

type azureAuth struct {
	db    database.Database
	store SessionStore
	azure auth.TokenParser
	jita  auth.TokenParser
	log   logrus.FieldLogger
}

func NewAuthenticator(azure auth.TokenParser, jita auth.TokenParser, db database.Database, store SessionStore, log logrus.FieldLogger) Authenticator {
	return &azureAuth{
		db:    db,
		store: store,
		azure: azure,
		jita:  jita,
		log:   log,
	}
}

func (s *azureAuth) ValidateJita(session *pb.Session, token string) error {
	user, err := s.jita.ParseString(token)
	if err != nil {
		return fmt.Errorf("failed to parse JITA token: %w", err)
	}

	if !strings.EqualFold(user.ID, session.ObjectID) {
		return fmt.Errorf("JITA token user ID (%s) does not match session user ID (%s)", user.ID, session.ObjectID)
	}

	return nil
}

func (s *azureAuth) Login(ctx context.Context, token, serial, platform string) (*pb.Session, error) {
	user, err := s.azure.ParseString(token)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	device, err := s.db.ReadDeviceBySerialPlatform(ctx, serial, platform)
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

	err = s.store.Set(ctx, session)
	if err != nil {
		s.log.WithError(err).WithField("device", device).WithField("session", session).Error("persist session")
		return nil, fmt.Errorf("persist session: %w", err)
	}

	return session, nil
}
