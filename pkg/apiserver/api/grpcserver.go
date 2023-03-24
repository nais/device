package api

import (
	"errors"
	"sync"

	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/jita"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

type grpcServer struct {
	pb.UnimplementedAPIServerServer

	authenticator        auth.Authenticator
	adminAuth            auth.UsernamePasswordAuthenticator
	gatewayAuth          auth.UsernamePasswordAuthenticator
	prometheusAuth       auth.UsernamePasswordAuthenticator
	jita                 jita.Client
	deviceConfigStreams  map[string]pb.APIServer_GetDeviceConfigurationServer
	gatewayConfigStreams map[string]pb.APIServer_GetGatewayConfigurationServer
	gatewayLock          sync.RWMutex
	deviceLock           sync.RWMutex
	db                   database.APIServer
	triggerGatewaySync   chan<- struct{}
}

var _ pb.APIServerServer = &grpcServer{}

var ErrNoSession = errors.New("no session")

func NewGRPCServer(db database.APIServer, authenticator auth.Authenticator, adminAuth, gatewayAuth, prometheusAuth auth.UsernamePasswordAuthenticator, jita jita.Client, triggerGatewaySync chan<- struct{}) *grpcServer {
	return &grpcServer{
		deviceConfigStreams:  make(map[string]pb.APIServer_GetDeviceConfigurationServer),
		gatewayConfigStreams: make(map[string]pb.APIServer_GetGatewayConfigurationServer),
		authenticator:        authenticator,
		adminAuth:            adminAuth,
		gatewayAuth:          gatewayAuth,
		prometheusAuth:       prometheusAuth,
		db:                   db,
		jita:                 jita,
		triggerGatewaySync:   triggerGatewaySync,
	}
}

func (s *grpcServer) triggerGatewayConfigurationSync() {
	select {
	case s.triggerGatewaySync <- struct{}{}:
	default:
		log.Warn("dropped trigger gateway sync, channel full")
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
