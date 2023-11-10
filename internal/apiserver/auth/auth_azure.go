package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/auth"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/random"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type azureAuth struct {
	db    database.APIServer
	store SessionStore
	Azure *auth.Azure
}

func NewAuthenticator(azureConfig *auth.Azure, db database.APIServer, store SessionStore) Authenticator {
	return &azureAuth{
		db:    db,
		store: store,
		Azure: azureConfig,
	}
}

func (s *azureAuth) Login(ctx context.Context, token, serial, platform string) (*pb.Session, error) {
	parsedToken, err := jwt.ParseString(token, s.Azure.JwtOptions()...)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, err := parsedToken.AsMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("convert claims to map: %s", err)
	}

	var groups []string
	for _, group := range claims["groups"].([]any) {
		groups = append(groups, group.(string))
	}

	if !auth.UserInNaisdeviceApprovalGroup(claims) {
		return nil, fmt.Errorf("do's and don'ts not accepted, visit: https://naisdevice-approval.external.prod-gcp.nav.cloud.nais.io/ to read and accept")
	}

	username := claims["preferred_username"].(string)

	device, err := s.db.ReadDeviceBySerialPlatform(ctx, serial, platform)
	if err != nil {
		return nil, fmt.Errorf("read device (%s, %s), user: %s, err: %v", serial, platform, username, err)
	}

	if !strings.EqualFold(username, device.Username) {
		return nil, fmt.Errorf("username (%s) does not match device username (%s)", username, device.Username)
	}

	session := &pb.Session{
		Key:      random.RandomString(20, random.LettersAndNumbers),
		Expiry:   timestamppb.New(time.Now().Add(SessionDuration)),
		Groups:   groups,
		ObjectID: claims["oid"].(string),
		Device:   device,
	}

	err = s.store.Set(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("persist session: %s", err)
	}

	return session, nil
}
