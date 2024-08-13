package api

import (
	"context"
	"fmt"
	"time"

	"github.com/nais/device/internal/apiserver/jita"
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

	trigger, err := s.gateways.Add(request.Gateway)
	if err != nil {
		return status.Errorf(codes.Aborted, "this gateway already has an open session")
	}
	defer s.gateways.Remove(request.Gateway)

	log.Info("gateway connected")
	defer log.Info("gateway disconnected")

	metrics.SetGatewayConnected(request.Gateway, true)
	defer metrics.SetGatewayConnected(request.Gateway, false)

	updateGatewayTicker := time.NewTicker(10 * time.Second)
	defer updateGatewayTicker.Stop()

	var lastCfg *pb.GetGatewayConfigurationResponse
	for {
		if cfg, err := s.makeGatewayConfiguration(stream.Context(), request.Gateway); err != nil {
			log.WithError(err).Error("make gateway config")
		} else if equalGatewayConfigurations(lastCfg, cfg) {
			// no change, don't send
		} else {
			if err := stream.Send(cfg); err != nil {
				log.WithError(err).Error("send gateway config")
			} else {
				lastCfg = cfg
			}
		}

		// block until trigger or done
		select {
		case <-trigger:
		case <-updateGatewayTicker.C:
		case <-stream.Context().Done():
			return nil
		}
	}
}

// Return a list of user sessions that are authorized to access a gateway through JITA.
func (s grpcServer) privilegedUsersForGateway(gateway *pb.Gateway) []jita.PrivilegedUser {
	if s.jita == nil {
		return nil
	}

	privilegedUsers := s.jita.GetPrivilegedUsersForGateway(gateway.Name)

	m, _ := metrics.PrivilegedUsersPerGateway.GetMetricWithLabelValues(gateway.Name)
	m.Set(float64(len(privilegedUsers)))

	return privilegedUsers
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

	filters := []func(*pb.Session) bool{
		sessionForGatewayGroups(gateway.AccessGroupIDs),
		sessionIsHealthy,
	}

	if gateway.RequiresPrivilegedAccess {
		privilegedUsers := s.privilegedUsersForGateway(gateway)
		filters = append(filters, sessionIsPrivileged(privilegedUsers))
	}

	// Hack to prevent windows users from connecting to the ms-login gateway.
	if gateway.Name == GatewayMSLoginName {
		filters = append(filters, not(sessionIsPlatform("windows")))
	}

	sessions := filterList(s.sessionStore.All(), filters...)

	devices := make([]*pb.Device, len(sessions))
	for i, session := range sessions {
		devices[i] = session.GetDevice()
	}

	gatewayConfig := &pb.GetGatewayConfigurationResponse{
		Devices:    devices,
		RoutesIPv4: gateway.GetRoutesIPv4(),
		RoutesIPv6: gateway.GetRoutesIPv6(),
	}

	metrics.GatewayConfigsReturned.WithLabelValues(gateway.Name).Inc()

	return gatewayConfig, nil
}

func (s *grpcServer) SendAllGatewayConfigurations() {
	s.gateways.TriggerAll()
}
