package api

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/nais/device/internal/apiserver/kolide"
	apiserver_metrics "github.com/nais/device/internal/apiserver/metrics"
	"github.com/nais/device/internal/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *grpcServer) GetDeviceConfiguration(request *pb.GetDeviceConfigurationRequest, stream pb.APIServer_GetDeviceConfigurationServer) error {
	s.log.WithField("session", request.SessionKey).Debugf("get device configuration: started")
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

	issues := []*pb.DeviceIssue{}
	kolideDeviceStream := make(chan kolide.Device)
	if s.kolideClient != nil {
		kolideDevice, err := s.kolideClient.GetDevice(stream.Context(), session.GetDevice().GetUsername(), session.GetDevice().GetPlatform(), session.GetDevice().GetSerial())
		if err != nil {
			return err
		}
		issues = kolideDevice.Issues()

		watchDone := &sync.Mutex{}
		watchDone.Lock()
		go s.watchKolideDevice(stream.Context(), s.kolideClient, kolideDevice, kolideDeviceStream, watchDone)
		defer func() {
			watchDone.Lock()
			close(kolideDeviceStream)
		}()
	}

	if len(session.GetGroups()) == 0 {
		s.log.WithField("deviceId", session.GetDevice().GetId()).Warnf("session with no groups detected")
	}

	defer func() { // cleanup
		s.deviceConfigTriggerLock.Lock()
		delete(s.deviceConfigTrigger, deviceId)
		apiserver_metrics.DevicesConnected.Set(float64(len(s.deviceConfigTrigger)))
		s.deviceConfigTriggerLock.Unlock()
	}()

	timeout := time.After(time.Until(session.GetExpiry().AsTime()))

	for {
		select {
		case <-timeout:
			s.log.Debugf("session for device %d timed out, tearing down", deviceId)
			return nil
		case <-stream.Context().Done(): // Disconnect
			s.log.Debugf("stream context for device %d done, tearing down", deviceId)
			return nil
		case d := <-kolideDeviceStream:
			issues = d.Issues()
		case <-trigger: // Send config triggered
			session, err := s.sessionStore.Get(stream.Context(), request.SessionKey)
			if err != nil {
				return err
			}

			cfg, err := s.makeDeviceConfiguration(stream.Context(), session, issues)
			if err != nil {
				s.log.Errorf("make device config: %v", err)
			} else {
				err := stream.Send(cfg)
				if err != nil {
					s.log.Debugf("stream end for device %d failed, err: %v", deviceId, err)
				}
			}
		}
	}
}

func (s *grpcServer) watchKolideDevice(ctx context.Context, kolideClient kolide.Client, d kolide.Device, deviceStream chan<- kolide.Device, done *sync.Mutex) {
	defer done.Unlock()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d, err := kolideClient.GetDevice(ctx, d.AssignedOwner.Email, d.Platform, d.Serial)
			if err != nil {
				s.log.Errorf("get device from kolide: %v", err)
			} else {
				deviceStream <- d
			}
		}
	}
}

func (s *grpcServer) makeDeviceConfiguration(ctx context.Context, sessionInfo *pb.Session, issues []*pb.DeviceIssue) (*pb.GetDeviceConfigurationResponse, error) {
	device, err := s.db.ReadDeviceById(ctx, sessionInfo.GetDevice().GetId())
	if err != nil {
		return nil, fmt.Errorf("read device from db: %w", err)
	}

	if !device.GetHealthy() || slices.ContainsFunc(issues, AfterGracePeriod) {
		return &pb.GetDeviceConfigurationResponse{
			Status: pb.DeviceConfigurationStatus_DeviceUnhealthy,
			Issues: issues,
		}, nil
	}

	gateways, err := s.UserGateways(ctx, sessionInfo.GetGroups())
	if err != nil {
		return nil, fmt.Errorf("get user gateways: %w", err)
	}

	apiserver_metrics.DeviceConfigsReturned.WithLabelValues(device.Serial, device.Username).Inc()

	return &pb.GetDeviceConfigurationResponse{
		Status:   pb.DeviceConfigurationStatus_DeviceHealthy,
		Issues:   issues,
		Gateways: gateways,
	}, nil
}

func (s *grpcServer) SendDeviceConfiguration(device *pb.Device) {
	s.deviceConfigTriggerLock.RLock()
	defer s.deviceConfigTriggerLock.RUnlock()

	c, ok := s.deviceConfigTrigger[device.GetId()]
	if !ok {
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

	return filtered, nil
}

func (s *grpcServer) Login(ctx context.Context, r *pb.APIServerLoginRequest) (*pb.APIServerLoginResponse, error) {
	version := r.Version
	if version == "" {
		version = "unknown"
	}
	apiserver_metrics.LoginRequests.WithLabelValues(version).Inc()

	session, err := s.authenticator.Login(ctx, r.Token, r.Serial, r.Platform)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "login: %v", err)
	}

	s.SendAllGatewayConfigurations()

	return &pb.APIServerLoginResponse{
		Session: session,
	}, nil
}
