package api

import (
	"context"
	"sync"

	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/apiserver/jita"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
)

type grpcServer struct {
	pb.UnimplementedAPIServerServer

	authenticator  auth.Authenticator
	adminAuth      auth.UsernamePasswordAuthenticator
	gatewayAuth    auth.UsernamePasswordAuthenticator
	prometheusAuth auth.UsernamePasswordAuthenticator
	jita           jita.Client

	deviceConfigTrigger     map[int64]chan struct{}
	deviceConfigTriggerLock sync.RWMutex

	gatewayConfigTrigger     map[string]chan struct{}
	gatewayConfigTriggerLock sync.RWMutex

	db           database.APIServer
	sessionStore auth.SessionStore

	programContext context.Context

	log *logrus.Entry
}

var _ pb.APIServerServer = &grpcServer{}

func NewGRPCServer(ctx context.Context, log *logrus.Entry, db database.APIServer, authenticator auth.Authenticator, adminAuth, gatewayAuth, prometheusAuth auth.UsernamePasswordAuthenticator, jita jita.Client, sessionStore auth.SessionStore) *grpcServer {
	return &grpcServer{
		deviceConfigTrigger:  make(map[int64]chan struct{}),
		gatewayConfigTrigger: make(map[string]chan struct{}),
		authenticator:        authenticator,
		adminAuth:            adminAuth,
		gatewayAuth:          gatewayAuth,
		prometheusAuth:       prometheusAuth,
		db:                   db,
		jita:                 jita,
		sessionStore:         sessionStore,
		programContext:       ctx,
		log:                  log,
	}
}

func authenticateAny(username, password string, auths ...auth.UsernamePasswordAuthenticator) error {
	for _, a := range auths {
		if a.Authenticate(username, password) == nil {
			return nil
		}
	}
	return auth.ErrInvalidAuth
}
