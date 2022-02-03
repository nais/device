package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/nais/device/pkg/apiserver/jita"
	apiserver_metrics "github.com/nais/device/pkg/apiserver/metrics"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/database"
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
	streams              map[string]pb.APIServer_GetDeviceConfigurationServer
	gatewayConfigStreams map[string]pb.APIServer_GetGatewayConfigurationServer
	lock                 sync.RWMutex
	db                   database.APIServer
	triggerGatewaySync   chan<- struct{}
}

var _ pb.APIServerServer = &grpcServer{}

var ErrNoSession = errors.New("no session")

func NewGRPCServer(db database.APIServer, authenticator auth.Authenticator, apikeyAuthenticator, gatewayAuthenticator auth.UsernamePasswordAuthenticator, jita jita.Client, triggerGatewaySync chan<- struct{}) *grpcServer {
	return &grpcServer{
		streams:              make(map[string]pb.APIServer_GetDeviceConfigurationServer),
		gatewayConfigStreams: make(map[string]pb.APIServer_GetGatewayConfigurationServer),
		authenticator:        authenticator,
		apikeyAuthenticator:  apikeyAuthenticator,
		gatewayAuthenticator: gatewayAuthenticator,
		db:                   db,
		jita:                 jita,
		triggerGatewaySync:   triggerGatewaySync,
	}
}

func (s *grpcServer) GetDeviceConfiguration(request *pb.GetDeviceConfigurationRequest, stream pb.APIServer_GetDeviceConfigurationServer) error {
	s.lock.RLock()
	s.streams[request.SessionKey] = stream
	apiserver_metrics.DevicesConnected.Set(float64(len(s.streams)))
	s.lock.RUnlock()

	// send initial device configuration
	err := s.SendDeviceConfiguration(stream.Context(), request.SessionKey)
	if err != nil {
		log.Errorf("send initial device configuration: %s", err)
	}

	// wait for disconnect
	<-stream.Context().Done()

	s.lock.Lock()
	delete(s.streams, request.SessionKey)
	apiserver_metrics.DevicesConnected.Set(float64(len(s.streams)))
	s.lock.Unlock()

	return nil
}

func (s *grpcServer) SendDeviceConfiguration(ctx context.Context, sessionKey string) error {
	s.lock.RLock()
	stream, ok := s.streams[sessionKey]
	s.lock.RUnlock()
	if !ok {
		return ErrNoSession
	}

	sessionInfo, err := s.db.ReadSessionInfo(ctx, sessionKey)
	if err != nil {
		return err
	}

	device, err := s.db.ReadDeviceById(ctx, sessionInfo.GetDevice().GetId())
	if err != nil {
		return fmt.Errorf("read device from db: %w", err)
	}

	if !device.GetHealthy() {
		return stream.Send(&pb.GetDeviceConfigurationResponse{
			Status: pb.DeviceConfigurationStatus_DeviceUnhealthy,
		})
	}

	gateways, err := s.UserGateways(ctx, sessionInfo.Groups)
	if err != nil {
		return fmt.Errorf("get user gateways: %w", err)
	}

	m, err := apiserver_metrics.DeviceConfigsReturned.GetMetricWithLabelValues(device.Serial, device.Username)
	if err != nil {
		log.Errorf("BUG: get metric: %s", err)
	} else {
		m.Inc()
	}

	return stream.Send(&pb.GetDeviceConfigurationResponse{
		Status:   pb.DeviceConfigurationStatus_DeviceHealthy,
		Gateways: gateways,
	})
}

func (s *grpcServer) UserGateways(ctx context.Context, userGroups []string) ([]*pb.Gateway, error) {
	gateways, err := s.db.ReadGateways(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading gateways from db: %v", err)
	}

	var filtered []*pb.Gateway
	for _, gw := range gateways {
		if userIsAuthorized(gw.AccessGroupIDs, userGroups) {
			gw.PasswordHash = ""
			filtered = append(filtered, gw)
		}
	}

	return filtered, nil
}

func (s *grpcServer) triggerGatewayConfigurationSync() {
	s.triggerGatewaySync <- struct{}{}
}

func (s *grpcServer) Login(ctx context.Context, r *pb.APIServerLoginRequest) (*pb.APIServerLoginResponse, error) {
	session, err := s.authenticator.Login(ctx, r.Token, r.Serial, r.Platform)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "login: %v", err)
	}

	s.triggerGatewayConfigurationSync()

	return &pb.APIServerLoginResponse{
		Session: session,
	}, nil
}

func (s *grpcServer) ListGateways(request *pb.ListGatewayRequest, stream pb.APIServer_ListGatewaysServer) error {
	err := s.apikeyAuthenticator.Authenticate(AdminUsername, request.Password)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	gateways, err := s.db.ReadGateways(stream.Context())
	if err != nil {
		return status.Error(codes.Unavailable, err.Error())
	}
	for _, gw := range gateways {
		err = stream.Send(gw)
		if err != nil {
			return status.Error(codes.Aborted, err.Error())
		}
	}

	return nil
}

func (s *grpcServer) GetGatewayConfiguration(request *pb.GetGatewayConfigurationRequest, stream pb.APIServer_GetGatewayConfigurationServer) error {
	err := s.gatewayAuthenticator.Authenticate(request.Gateway, request.Password)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	s.lock.RLock()
	_, hasSession := s.gatewayConfigStreams[request.Gateway]
	s.lock.RUnlock()

	if hasSession {
		return status.Errorf(codes.Aborted, "this gateway already has an open session")
	}

	s.lock.Lock()
	s.gatewayConfigStreams[request.Gateway] = stream
	s.reportOnlineGateways()
	s.lock.Unlock()

	defer func() {
		s.lock.Lock()
		delete(s.gatewayConfigStreams, request.Gateway)
		s.reportOnlineGateways()
		s.lock.Unlock()
	}()

	// send initial device configuration
	s.lock.RLock()
	err = s.SendGatewayConfiguration(stream.Context(), request.Gateway)
	s.lock.RUnlock()
	if err != nil {
		return fmt.Errorf("send initial gateway configuration: %s", err)
	}

	// wait for disconnect
	<-stream.Context().Done()

	return nil
}

func (s *grpcServer) SendAllGatewayConfigurations(ctx context.Context) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for gateway := range s.gatewayConfigStreams {
		err := s.SendGatewayConfiguration(ctx, gateway)
		if err != nil {
			return fmt.Errorf("send gateway config: %w", err)
		}
	}
	return nil
}

func (s *grpcServer) SendGatewayConfiguration(ctx context.Context, gatewayName string) error {
	stream, ok := s.gatewayConfigStreams[gatewayName]
	if !ok {
		return ErrNoSession
	}

	sessionInfos, err := s.db.ReadSessionInfos(ctx)
	if err != nil {
		return fmt.Errorf("read session infos from database: %w", err)
	}

	gateway, err := s.db.ReadGateway(ctx, gatewayName)
	if err != nil {
		return fmt.Errorf("read gateway from database: %w", err)
	}

	gatewayConfig := &pb.GetGatewayConfigurationResponse{
		Devices: healthy(
			authorized(
				gateway.AccessGroupIDs, privileged(s.jita, gateway, sessionInfos),
			),
		),
		Routes: gateway.Routes,
	}

	m, err := apiserver_metrics.GatewayConfigsReturned.GetMetricWithLabelValues(gateway.Name)
	if err != nil {
		log.Errorf("getting metric metric: %v", err)
	} else {
		m.Inc()
	}

	return stream.Send(gatewayConfig)
}

func (s *grpcServer) onlineGateways() []string {
	gateways := make([]string, 0, len(s.gatewayConfigStreams))
	for k := range s.gatewayConfigStreams {
		gateways = append(gateways, k)
	}
	return gateways
}

func (s *grpcServer) reportOnlineGateways() {
	apiserver_metrics.SetConnectedGateways(s.onlineGateways())
	log.Infof("Online gateways: %s", strings.Join(s.onlineGateways(), ", "))
}
