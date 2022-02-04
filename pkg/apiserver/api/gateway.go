package api

import (
	"context"
	"fmt"
	"strings"

	apiserver_metrics "github.com/nais/device/pkg/apiserver/metrics"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

	log.Infof("Gateway %s", strings.Join(s.onlineGateways(), ", "))

	defer func() {
		s.lock.Lock()
		delete(s.gatewayConfigStreams, request.Gateway)
		s.reportOnlineGateways()
		s.lock.Unlock()
	}()

	// send initial device configuration
	s.lock.RLock()
	err = s.SendInitialGatewayConfiguration(stream.Context(), request.Gateway)
	s.lock.RUnlock()
	if err != nil {
		return fmt.Errorf("send initial gateway configuration: %s", err)
	}

	// wait for disconnect
	<-stream.Context().Done()

	return nil
}

func (s *grpcServer) SendInitialGatewayConfiguration(ctx context.Context, gatewayName string) error {
	sessionInfos, err := s.db.ReadSessionInfos(ctx)
	if err != nil {
		return fmt.Errorf("read session infos from database: %w", err)
	}

	return s.SendGatewayConfiguration(ctx, gatewayName, sessionInfos)
}

func (s *grpcServer) SendAllGatewayConfigurations(ctx context.Context) error {
	s.lock.RLock()
	defer s.lock.RUnlock()

	sessionInfos, err := s.db.ReadSessionInfos(ctx)
	if err != nil {
		return fmt.Errorf("read session infos from database: %w", err)
	}

	for gateway := range s.gatewayConfigStreams {
		err := s.SendGatewayConfiguration(ctx, gateway, sessionInfos)
		if err != nil {
			return fmt.Errorf("send gateway config: %w", err)
		}
	}
	return nil
}

func (s *grpcServer) SendGatewayConfiguration(ctx context.Context, gatewayName string, sessionInfos []*pb.Session) error {
	stream, ok := s.gatewayConfigStreams[gatewayName]
	if !ok {
		return ErrNoSession
	}

	gateway, err := s.db.ReadGateway(ctx, gatewayName)
	if err != nil {
		return fmt.Errorf("read gateway from database: %w", err)
	}

	gatewayPrivilegedDevices := privileged(s.jita, gateway, sessionInfos)
	authorizedDevices := authorized(gateway.AccessGroupIDs, gatewayPrivilegedDevices)
	healthyDevices := healthy(authorizedDevices)
	gatewayConfig := &pb.GetGatewayConfigurationResponse{
		Devices: healthyDevices,
		Routes:  gateway.Routes,
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
}
