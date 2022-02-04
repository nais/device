package api

import (
	"errors"
	"sync"

	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/jita"
	"github.com/nais/device/pkg/pb"
)

const (
	AdminUsername = "admin"
)

type grpcServer struct {
	pb.UnimplementedAPIServerServer

	authenticator        auth.Authenticator
	apikeyAuthenticator  auth.UsernamePasswordAuthenticator
	gatewayAuthenticator auth.UsernamePasswordAuthenticator
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

func NewGRPCServer(db database.APIServer, authenticator auth.Authenticator, apikeyAuthenticator, gatewayAuthenticator auth.UsernamePasswordAuthenticator, jita jita.Client, triggerGatewaySync chan<- struct{}) *grpcServer {
	return &grpcServer{
		deviceConfigStreams:  make(map[string]pb.APIServer_GetDeviceConfigurationServer),
		gatewayConfigStreams: make(map[string]pb.APIServer_GetGatewayConfigurationServer),
		authenticator:        authenticator,
		apikeyAuthenticator:  apikeyAuthenticator,
		gatewayAuthenticator: gatewayAuthenticator,
		db:                   db,
		jita:                 jita,
		triggerGatewaySync:   triggerGatewaySync,
	}
}

func (s *grpcServer) triggerGatewayConfigurationSync() {
	s.triggerGatewaySync <- struct{}{}
}
