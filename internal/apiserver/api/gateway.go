package api

import (
	"context"
	"fmt"
	"time"

	"github.com/nais/device/internal/apiserver/metrics"
	"github.com/nais/device/internal/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *grpcServer) GetGatewayConfiguration(request *pb.GetGatewayConfigurationRequest, stream pb.APIServer_GetGatewayConfigurationServer) error {
	err := s.gatewayAuth.Authenticate(stream.Context(), request.Gateway, request.Password)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}

	log := s.log.WithField("gateway", request.Gateway)
	log.Info("incoming gateway connection")

	trigger, err := s.gateways.Add(request.Gateway)
	if err != nil {
		return status.Errorf(codes.Aborted, "this gateway already has an open session")
	}
	defer s.gateways.Remove(request.Gateway)

	log.Info("gateway connected", request.Gateway)
	metrics.SetGatewayConnected(request.Gateway, true)
	defer metrics.SetGatewayConnected(request.Gateway, false)

	updateGatewayTicker := time.NewTicker(10 * time.Second)
	var lastCfg *pb.GetGatewayConfigurationResponse
	for {
		if cfg, err := s.makeGatewayConfiguration(stream.Context(), request.Gateway); err != nil {
			log.WithError(err).Error("make gateway config")
		} else if equalGatewayConfigurations(lastCfg, cfg) {
			// no change, don't send
		} else {
			if err := stream.Send(cfg); err != nil {
				log.WithError(err).Error("send gateway config")
			}
			lastCfg = cfg
		}

		// block until trigger or done
		select {
		case <-trigger:
		case <-updateGatewayTicker.C:
		case <-stream.Context().Done():
			log.Info("gateway disconnected")
			return nil
		}
	}
}

func equalGatewayConfigurations(a, b *pb.GetGatewayConfigurationResponse) bool {
	if a == b {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a.Devices) != len(b.Devices) {
		return false
	}

	for i := range a.Devices {
		da, db := a.Devices[i], b.Devices[i]
		if da.Id != db.Id || da.Healthy() != db.Healthy() {
			return false
		}
	}

	for i := range a.RoutesIPv4 {
		if a.RoutesIPv4[i] != b.RoutesIPv4[i] {
			return false
		}
	}

	for i := range a.RoutesIPv6 {
		if a.RoutesIPv6[i] != b.RoutesIPv6[i] {
			return false
		}
	}
	return true
}

func (s *grpcServer) makeGatewayConfiguration(ctx context.Context, gatewayName string) (*pb.GetGatewayConfigurationResponse, error) {
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

	metrics.GatewayConfigsReturned.WithLabelValues(gateway.Name).Inc()

	return gatewayConfig, nil
}

func (s *grpcServer) SendAllGatewayConfigurations() {
	s.gateways.TriggerAll()
}
