package api

import (
	"errors"
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

	deviceConfigStreams     map[string]pb.APIServer_GetDeviceConfigurationServer
	deviceConfigStreamsLock sync.RWMutex

	gatewayConfigTrigger     map[string]chan struct{}
	gatewayConfigTriggerLock sync.RWMutex

	db           database.APIServer
	sessionStore auth.SessionStore
}

var _ pb.APIServerServer = &grpcServer{}

var ErrNoSession = errors.New("no session")

func TriggerGatewaySync(channel chan<- struct{}) {
	select {
	case channel <- struct{}{}:
	default:
	}
}

func NewGRPCServer(db database.APIServer, authenticator auth.Authenticator, adminAuth, gatewayAuth, prometheusAuth auth.UsernamePasswordAuthenticator, jita jita.Client, sessionStore auth.SessionStore) *grpcServer {
	return &grpcServer{
		deviceConfigStreams:  make(map[string]pb.APIServer_GetDeviceConfigurationServer),
		gatewayConfigTrigger: make(map[string]chan struct{}),
		authenticator:        authenticator,
		adminAuth:            adminAuth,
		gatewayAuth:          gatewayAuth,
		prometheusAuth:       prometheusAuth,
		db:                   db,
		jita:                 jita,
		sessionStore:         sessionStore,
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
