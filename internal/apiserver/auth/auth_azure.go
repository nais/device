package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/auth"
	"github.com/nais/device/internal/random"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type azureAuth struct {
	db    database.Database
	store SessionStore
	Azure *auth.Azure
	log   logrus.FieldLogger
}

func NewAuthenticator(azureConfig *auth.Azure, db database.Database, store SessionStore, log logrus.FieldLogger) Authenticator {
	return &azureAuth{
		db:    db,
		store: store,
		Azure: azureConfig,
		log:   log,
	}
}

func (s *azureAuth) Login(ctx context.Context, token, serial, platform string) (*pb.Session, error) {
	parsedToken, err := jwt.ParseString(token, s.Azure.JwtOptions()...)
	if err != nil {
		return nil, &ParseTokenError{err}
	}

	var groups []string
	if err := parsedToken.Get("groups", &groups); err != nil {
		return nil, fmt.Errorf("get groups from claims: %s", err)
	}

	if !auth.UserInNaisdeviceApprovalGroup(groups) {
		return nil, ErrTermsNotAccepted
	}

	username, err := auth.StringClaim("preferred_username", parsedToken)
	if err != nil {
		return nil, fmt.Errorf("missing claim: %w", err)
	}

	device, err := s.db.ReadDeviceBySerialPlatform(ctx, serial, platform)
	if err != nil {
		return nil, fmt.Errorf("read device (%s, %s), user: %s, err: %v", serial, platform, username, err)
	}

	if !strings.EqualFold(username, device.Username) {
		return nil, fmt.Errorf("username (%s) does not match device username (%s)", username, device.Username)
	}

	oid, err := auth.StringClaim("oid", parsedToken)
	if err != nil {
		return nil, fmt.Errorf("missing claim: %w", err)
	}

	session := &pb.Session{
		Key:      random.RandomString(20, random.LettersAndNumbers),
		Expiry:   timestamppb.New(time.Now().Add(SessionDuration)),
		Groups:   groups,
		ObjectID: oid,
		Device:   device,
	}

	err = s.store.Set(ctx, session)
	if err != nil {
		s.log.WithError(err).WithField("device", device).WithField("session", session).Error("persist session")
		return nil, fmt.Errorf("persist session: %w", err)
	}

	return session, nil
}
