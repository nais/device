package api

import (
	"context"
	"fmt"

	apiserver_metrics "github.com/nais/device/internal/apiserver/metrics"
	"github.com/nais/device/internal/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *grpcServer) GetGatewayConfiguration(request *pb.GetGatewayConfigurationRequest, stream pb.APIServer_GetGatewayConfigurationServer) error {
	err := s.gatewayAuth.Authenticate(request.Gateway, request.Password)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	s.gatewayConfigTriggerLock.RLock()
	_, hasSession := s.gatewayConfigTrigger[request.Gateway]
	s.gatewayConfigTriggerLock.RUnlock()

	if hasSession {
		return status.Errorf(codes.Aborted, "this gateway already has an open session")
	}

	c := make(chan struct{}, 1)
	c <- struct{}{} // trigger config send immediately

	s.gatewayConfigTriggerLock.Lock()
	s.gatewayConfigTrigger[request.Gateway] = c
	s.log.Infof("Gateway %s connected (%d active gateways)", request.Gateway, len(s.gatewayConfigTrigger))
	s.gatewayConfigTriggerLock.Unlock()

	for {
		select {
		case <-c:
			cfg, err := s.MakeGatewayConfiguration(stream.Context(), request.Gateway)
			if err != nil {
				s.log.Errorf("make gateway config: %v", err)
			}

			err = stream.Send(cfg)
			if err != nil {
				s.log.Errorf("send gateway config: %v", err)
			}

		case <-stream.Context().Done():
			s.gatewayConfigTriggerLock.Lock()
			delete(s.gatewayConfigTrigger, request.Gateway)
			s.log.Infof("Gateway %s disconnected (%d active gateways)", request.Gateway, len(s.gatewayConfigTrigger))
			s.gatewayConfigTriggerLock.Unlock()

			return nil
		}
	}
}

func (s *grpcServer) SendAllGatewayConfigurations() {
	s.gatewayConfigTriggerLock.RLock()
	defer s.gatewayConfigTriggerLock.RUnlock()

	for _, c := range s.gatewayConfigTrigger {
		select {
		case c <- struct{}{}:
		default:
		}
	}
}

func (s *grpcServer) MakeGatewayConfiguration(ctx context.Context, gatewayName string) (*pb.GetGatewayConfigurationResponse, error) {
	gateway, err := s.db.ReadGateway(ctx, gatewayName)
	if err != nil {
		return nil, fmt.Errorf("read gateway from database: %w", err)
	}

	allSessions := s.sessionStore.All()
	privilegedDevices := privileged(s.jita, gateway, allSessions)
	authorizedDevices := authorized(gateway.AccessGroupIDs, privilegedDevices)
	healthyDevices := healthy(authorizedDevices)
	uniqueDevices := unique(healthyDevices)

	gatewayConfig := &pb.GetGatewayConfigurationResponse{
		Devices:    uniqueDevices,
		RoutesIPv4: gateway.GetRoutesIPv4(),
		RoutesIPv6: gateway.GetRoutesIPv6(),
	}

	apiserver_metrics.GatewayConfigsReturned.WithLabelValues(gateway.Name).Inc()

	return gatewayConfig, nil
}

func (s *grpcServer) ReportOnlineGateways() {
	s.gatewayConfigTriggerLock.RLock()
	connectedGatewayNames := make([]string, 0, len(s.gatewayConfigTrigger))
	for k := range s.gatewayConfigTrigger {
		connectedGatewayNames = append(connectedGatewayNames, k)
	}
	s.gatewayConfigTriggerLock.RUnlock()

	allGatewayNames, err := s.getAllGatewayNames()
	if err != nil {
		s.log.Errorf("unable to report online gateways: %v", err)
		return
	}

	apiserver_metrics.SetConnectedGateways(allGatewayNames, connectedGatewayNames)
}

func (s *grpcServer) getAllGatewayNames() ([]string, error) {
	allGateways, err := s.db.ReadGateways(s.programContext)
	allGatewayNames := make([]string, len(allGateways))
	if err != nil {
		return nil, fmt.Errorf("read gateways from database: %w", err)
	}

	for i := range allGateways {
		allGatewayNames[i] = allGateways[i].Name
	}

	return allGatewayNames, nil
}
