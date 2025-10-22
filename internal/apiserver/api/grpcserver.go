package api

import (
	"context"

	"github.com/nais/device/internal/apiserver/api/triggers"
	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
)

type grpcServer struct {
	pb.UnimplementedAPIServerServer

	authenticator  auth.Authenticator
	adminAuth      auth.UsernamePasswordAuthenticator
	gatewayAuth    auth.UsernamePasswordAuthenticator
	prometheusAuth auth.UsernamePasswordAuthenticator
	kolideClient   kolide.Client
	kolideEnabled  bool

	devices  *triggers.StreamTriggers[int64]
	gateways *triggers.StreamTriggers[string]

	db           database.Database
	sessionStore auth.SessionStore

	programContext context.Context

	log logrus.FieldLogger
}

var _ pb.APIServerServer = &grpcServer{}

func NewGRPCServer(ctx context.Context, log logrus.FieldLogger, db database.Database, authenticator auth.Authenticator, adminAuth, gatewayAuth, prometheusAuth auth.UsernamePasswordAuthenticator, sessionStore auth.SessionStore, kolideClient kolide.Client, kolideEnabled bool) *grpcServer {
	return &grpcServer{
		devices:        triggers.New[int64](),
		gateways:       triggers.New[string](),
		authenticator:  authenticator,
		adminAuth:      adminAuth,
		gatewayAuth:    gatewayAuth,
		prometheusAuth: prometheusAuth,
		db:             db,
		kolideClient:   kolideClient,
		sessionStore:   sessionStore,
		programContext: ctx,
		log:            log,
		kolideEnabled:  kolideEnabled,
	}
}

func authenticateAny(ctx context.Context, username, password string, auths ...auth.UsernamePasswordAuthenticator) error {
	for _, a := range auths {
		if a.Authenticate(ctx, username, password) == nil {
			return nil
		}
	}

	return auth.ErrInvalidAuth
}

func (s *grpcServer) UpdateKolideChecks(ctx context.Context) error {
	if checks, err := s.kolideClient.GetChecks(ctx); err != nil {
		return err
	} else if err = s.db.UpdateKolideChecks(ctx, checks); err != nil {
		return err
	}
	return nil
}
