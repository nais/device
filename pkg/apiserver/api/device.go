package api

import (
	"context"
	"fmt"
	"time"

	apiserver_metrics "github.com/nais/device/pkg/apiserver/metrics"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *grpcServer) GetDeviceConfiguration(request *pb.GetDeviceConfigurationRequest, stream pb.APIServer_GetDeviceConfigurationServer) error {
	log.WithField("session", request.SessionKey).Debugf("get device configuration: started")
	trigger := make(chan struct{}, 1)
	trigger <- struct{}{} // immediately send one config back on the stream

	session, err := s.sessionStore.Get(stream.Context(), request.SessionKey)
	if err != nil {
		return err
	}
	deviceId := session.GetDevice().GetId()

	s.deviceConfigTriggerLock.Lock()
	s.deviceConfigTrigger[deviceId] = trigger
	apiserver_metrics.DevicesConnected.Set(float64(len(s.deviceConfigTrigger)))
	s.deviceConfigTriggerLock.Unlock()

	if len(session.GetGroups()) == 0 {
		log.WithField("deviceId", session.GetDevice().GetId()).Warnf("session with no groups detected")
	}

	defer func() {
		s.deviceConfigTriggerLock.Lock()
		delete(s.deviceConfigTrigger, deviceId)
		apiserver_metrics.DevicesConnected.Set(float64(len(s.deviceConfigTrigger)))
		s.deviceConfigTriggerLock.Unlock()
	}()

	timeout := time.After(time.Until(session.GetExpiry().AsTime()))

	for {
		select {
		case <-timeout:
			log.Debugf("session for device %d timed out, tearing down", deviceId)
			return nil
		case <-stream.Context().Done(): // Disconnect
			log.Debugf("stream context for device %d done, tearing down", deviceId)
			return nil
		case <-trigger: // Send config triggered
			session, err := s.sessionStore.Get(stream.Context(), request.SessionKey)
			if err != nil {
				return err
			}

			cfg, err := s.makeDeviceConfiguration(stream.Context(), session)
			if err != nil {
				log.Errorf("make device config: %v", err)
			} else {
				err := stream.Send(cfg)
				if err != nil {
					log.Debugf("stream end for device %d failed, err: %v", deviceId, err)
				}
			}
		}
	}
}

func (s *grpcServer) makeDeviceConfiguration(ctx context.Context, sessionInfo *pb.Session) (*pb.GetDeviceConfigurationResponse, error) {
	device, err := s.db.ReadDeviceById(ctx, sessionInfo.GetDevice().GetId())
	if err != nil {
		return nil, fmt.Errorf("read device from db: %w", err)
	}

	if !device.GetHealthy() {
		return &pb.GetDeviceConfigurationResponse{Status: pb.DeviceConfigurationStatus_DeviceUnhealthy}, nil
	}

	gateways, err := s.UserGateways(ctx, sessionInfo.GetGroups())
	if err != nil {
		return nil, fmt.Errorf("get user gateways: %w", err)
	}

	apiserver_metrics.DeviceConfigsReturned.WithLabelValues(device.Serial, device.Username).Inc()

	return &pb.GetDeviceConfigurationResponse{
		Status:   pb.DeviceConfigurationStatus_DeviceHealthy,
		Gateways: gateways,
	}, nil
}

func (s *grpcServer) SendDeviceConfiguration(device *pb.Device) {
	s.deviceConfigTriggerLock.RLock()
	defer s.deviceConfigTriggerLock.RUnlock()

	c, ok := s.deviceConfigTrigger[device.GetId()]
	if !ok {
		log.Errorf("send device config: no active stream found")
		return
	}

	select {
	case c <- struct{}{}:
	default:
	}
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

	s.SendAllGatewayConfigurations()

	return &pb.APIServerLoginResponse{
		Session: session,
	}, nil
}
