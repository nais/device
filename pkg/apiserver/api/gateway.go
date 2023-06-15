package api

import (
	"context"
	"fmt"

	apiserver_metrics "github.com/nais/device/pkg/apiserver/metrics"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *grpcServer) GetGatewayConfiguration(request *pb.GetGatewayConfigurationRequest, stream pb.APIServer_GetGatewayConfigurationServer) error {
	err := s.gatewayAuth.Authenticate(request.Gateway, request.Password)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	s.gatewayLock.RLock()
	_, hasSession := s.gatewayConfigStreams[request.Gateway]
	s.gatewayLock.RUnlock()

	if hasSession {
		return status.Errorf(codes.Aborted, "this gateway already has an open session")
	}

	s.gatewayLock.Lock()
	s.gatewayConfigStreams[request.Gateway] = stream
	s.reportOnlineGateways()
	log.Infof("Gateway %s connected (%d active gateways)", request.Gateway, len(s.gatewayConfigStreams))
	s.gatewayLock.Unlock()

	defer func() {
		s.gatewayLock.Lock()
		delete(s.gatewayConfigStreams, request.Gateway)
		s.reportOnlineGateways()
		s.gatewayLock.Unlock()
	}()

	// send initial device configuration
	s.gatewayLock.RLock()
	err = s.SendInitialGatewayConfiguration(stream.Context(), request.Gateway)
	s.gatewayLock.RUnlock()
	if err != nil {
		return fmt.Errorf("send initial gateway configuration: %s", err)
	}

	// wait for disconnect
	<-stream.Context().Done()

	log.Infof("Gateway %s disconnected (%d active gateways)", request.Gateway, len(s.gatewayConfigStreams))

	return nil
}

func (s *grpcServer) SendInitialGatewayConfiguration(ctx context.Context, gatewayName string) error {
	log.Infof("sending initial gateway config to %s", gatewayName)
	defer func() { log.Infof("done sending initial gateway config to %s", gatewayName) }()

	sessionInfos, err := s.db.ReadSessionInfos(ctx)
	if err != nil {
		return fmt.Errorf("read session infos from database: %w", err)
	}

	return s.SendGatewayConfiguration(ctx, gatewayName, sessionInfos)
}

func (s *grpcServer) SendAllGatewayConfigurations(ctx context.Context) error {
	log.Info("sending all gateway configs")
	defer func() { log.Info("done sending all gateway configs") }()

	s.gatewayLock.RLock()
	defer s.gatewayLock.RUnlock()

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
	log.Infof("sending gateway config to %s", gatewayName)
	defer func() { log.Infof("done sending gateway config to %s", gatewayName) }()

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
