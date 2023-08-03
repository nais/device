package api

import (
	"context"
	"sync"

	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/jita"
	"github.com/nais/device/pkg/pb"
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
}

var _ pb.APIServerServer = &grpcServer{}

func NewGRPCServer(ctx context.Context, db database.APIServer, authenticator auth.Authenticator, adminAuth, gatewayAuth, prometheusAuth auth.UsernamePasswordAuthenticator, jita jita.Client, sessionStore auth.SessionStore) *grpcServer {
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
