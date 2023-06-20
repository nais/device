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

func (s *grpcServer) GetDeviceConfiguration(request *pb.GetDeviceConfigurationRequest, stream pb.APIServer_GetDeviceConfigurationServer) error {
	log.WithField("session", request.SessionKey).Debugf("get device configuration: started")
	s.deviceConfigStreamsLock.Lock()
	s.deviceConfigStreams[request.SessionKey] = stream
	apiserver_metrics.DevicesConnected.Set(float64(len(s.deviceConfigStreams)))
	s.deviceConfigStreamsLock.Unlock()

	// send initial device configuration
	err := s.SendDeviceConfiguration(stream.Context(), request.SessionKey)
	if err != nil {
		log.Errorf("send initial device configuration: %s", err)
	}
	log.WithField("session", request.SessionKey).Debugf("get device configuration: sent initial")

	// wait for disconnect
	<-stream.Context().Done()
	log.WithField("session", request.SessionKey).Debugf("get device configuration: finished")

	s.deviceConfigStreamsLock.Lock()
	delete(s.deviceConfigStreams, request.SessionKey)
	apiserver_metrics.DevicesConnected.Set(float64(len(s.deviceConfigStreams)))
	s.deviceConfigStreamsLock.Unlock()

	log.WithField("session", request.SessionKey).Debugf("get device configuration: cleaned up stream map")

	return nil
}

func (s *grpcServer) SendDeviceConfiguration(ctx context.Context, sessionKey string) error {
	log.WithField("session", sessionKey).Debugf("send device configuration: started")
	s.deviceConfigStreamsLock.RLock()
	stream, ok := s.deviceConfigStreams[sessionKey]
	s.deviceConfigStreamsLock.RUnlock()
	if !ok {
		return ErrNoSession
	}

	sessionInfo, err := s.db.ReadSessionInfo(ctx, sessionKey)
	if err != nil {
		return err
	}

	if len(sessionInfo.GetGroups()) == 0 {
		log.WithField("deviceId", sessionInfo.GetDevice().GetId()).Warnf("session with no groups detected")
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

	gateways, err := s.UserGateways(ctx, sessionInfo.GetGroups())
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
		if StringSliceHasIntersect(gw.AccessGroupIDs, userGroups) {
			gw.PasswordHash = ""
			filtered = append(filtered, gw)
		}
	}

	if len(filtered) == 0 {
		var gwIds map[string][]string
		for _, gw := range gateways {
			gwIds[gw.GetName()] = gw.GetAccessGroupIDs()
		}
		log.Warnf("returning empty filtered gateway list for userGroups: %+v and gateways: %+v", userGroups, gwIds)
	}

	return filtered, nil
}

func (s *grpcServer) Login(ctx context.Context, r *pb.APIServerLoginRequest) (*pb.APIServerLoginResponse, error) {
	session, err := s.authenticator.Login(ctx, r.Token, r.Serial, r.Platform)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "login: %v", err)
	}

	s.SendAllGatewayConfigurations(ctx)

	return &pb.APIServerLoginResponse{
		Session: session,
	}, nil
}
