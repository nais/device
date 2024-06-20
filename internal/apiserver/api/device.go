package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	apiserver_metrics "github.com/nais/device/internal/apiserver/metrics"
	"github.com/nais/device/internal/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *grpcServer) GetDeviceConfiguration(request *pb.GetDeviceConfigurationRequest, stream pb.APIServer_GetDeviceConfigurationServer) error {
	s.log.WithField("session", request.SessionKey).Debugf("get device configuration: started")
	trigger := make(chan struct{}, 1)

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
		s.log.WithField("deviceId", deviceId).Warnf("session with no groups detected")
	}

	defer func() { // cleanup
		s.deviceConfigTriggerLock.Lock()
		delete(s.deviceConfigTrigger, deviceId)
		apiserver_metrics.DevicesConnected.Set(float64(len(s.deviceConfigTrigger)))
		s.deviceConfigTriggerLock.Unlock()
	}()

	timeout := time.After(time.Until(session.GetExpiry().AsTime()))
	updateDeviceTicker := time.NewTicker(1 * time.Minute)
	var lastCfg *pb.GetDeviceConfigurationResponse
	for {
		cfg, err := s.makeDeviceConfiguration(stream.Context(), request.SessionKey)
		if err != nil {
			s.log.WithError(err).Error("make device config")
		} else if EqualDeviceConfigurations(lastCfg, cfg) {
			// no change, don't send
		} else {
			err := stream.Send(cfg)
			if err != nil {
				s.log.WithError(err).Debugf("stream send for device %+v failed", cfg)
			}
			lastCfg = cfg
		}

		// block until trigger or done
		select {
		case <-timeout:
			s.log.Debugf("session for device %d timed out, tearing down", deviceId)
			return nil
		case <-stream.Context().Done(): // Disconnect
			s.log.Debugf("stream context for device %d done, tearing down", deviceId)
			return nil
		case <-updateDeviceTicker.C:
		case <-trigger:
		}
	}
}

func EqualDeviceConfigurations(a, b *pb.GetDeviceConfigurationResponse) bool {
	if a == b {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if a.Status != b.Status {
		return false
	}

	if len(a.Issues) != len(b.Issues) {
		return false
	}

	for i, issue := range a.Issues {
		if issue != b.Issues[i] {
			return false
		}
	}

	if len(a.Gateways) != len(b.Gateways) {
		return false
	}

	if len(a.Issues) != len(b.Issues) {
		return false
	}

	for i := range a.Gateways {
		if !a.GetGateways()[i].Equal(b.GetGateways()[i]) {
			return false
		}
	}

	for i := range a.Issues {
		if !a.GetIssues()[i].Equal(b.GetIssues()[i]) {
			return false
		}
	}

	return true
}

func (s *grpcServer) makeDeviceConfiguration(ctx context.Context, sessionKey string) (*pb.GetDeviceConfigurationResponse, error) {
	session, err := s.sessionStore.Get(ctx, sessionKey)
	if err != nil {
		return nil, err
	}

	device, err := s.db.ReadDeviceById(ctx, session.GetDevice().GetId())
	if err != nil {
		return nil, err
	}

	if !device.Healthy() {
		return &pb.GetDeviceConfigurationResponse{
			Status: pb.DeviceConfigurationStatus_DeviceUnhealthy,
			Issues: device.Issues,
		}, nil
	}

	gateways, err := s.UserGateways(ctx, session.GetGroups())
	if err != nil {
		return nil, fmt.Errorf("get user gateways: %w", err)
	}

	apiserver_metrics.DeviceConfigsReturned.WithLabelValues(device.Serial, device.Username).Inc()

	return &pb.GetDeviceConfigurationResponse{
		Status:   pb.DeviceConfigurationStatus_DeviceHealthy,
		Issues:   device.Issues,
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

func (s *grpcServer) UpdateAllDevices(ctx context.Context) error {
	devices, err := s.db.ReadDevices(ctx)
	if err != nil {
		return nil
	}

	err = s.kolideClient.FillKolideData(ctx, devices)
	if err != nil {
		return err
	}

	err = s.db.UpdateDevices(ctx, devices)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.log.Errorf("storing device: %v", err)
	}

	for _, device := range devices {
		s.SendDeviceConfiguration(device)
	}

	return err
}
