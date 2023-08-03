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
	log.Infof("Gateway %s connected (%d active gateways)", request.Gateway, len(s.gatewayConfigTrigger))
	s.gatewayConfigTriggerLock.Unlock()

	s.reportOnlineGateways()

	for {
		select {
		case <-c:
			cfg, err := s.MakeGatewayConfiguration(stream.Context(), request.Gateway)
			if err != nil {
				log.Errorf("make gateway config: %v", err)
			}

			log.Infof("sending gateway config to %s", request.Gateway)
			err = stream.Send(cfg)
			if err != nil {
				log.Errorf("send gateway config: %v", err)
			} else {
				log.Infof("sent gateway config to %s", request.Gateway)
			}

		case <-stream.Context().Done():
			s.gatewayConfigTriggerLock.Lock()
			delete(s.gatewayConfigTrigger, request.Gateway)
			log.Infof("Gateway %s disconnected (%d active gateways)", request.Gateway, len(s.gatewayConfigTrigger))
			s.gatewayConfigTriggerLock.Unlock()

			s.reportOnlineGateways()

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
	log.Infof("sending gateway config to %s", gatewayName)
	defer func() { log.Infof("done sending gateway config to %s", gatewayName) }()

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
		Devices: uniqueDevices,
		Routes:  gateway.Routes,
	}

	apiserver_metrics.GatewayConfigsReturned.WithLabelValues(gateway.Name).Inc()

	return gatewayConfig, nil
}

func (s *grpcServer) reportOnlineGateways() {
	s.gatewayConfigTriggerLock.RLock()
	connectedGatewayNames := make([]string, 0, len(s.gatewayConfigTrigger))
	for k := range s.gatewayConfigTrigger {
		connectedGatewayNames = append(connectedGatewayNames, k)
	}
	s.gatewayConfigTriggerLock.RUnlock()

	allGatewayNames, err := s.getAllGatewayNames()
	if err != nil {
		log.Errorf("unable to report online gateways: %v", err)
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
