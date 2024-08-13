package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nais/device/internal/apiserver/metrics"
	"github.com/nais/device/internal/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *grpcServer) GetDeviceConfiguration(request *pb.GetDeviceConfigurationRequest, stream pb.APIServer_GetDeviceConfigurationServer) error {
	session, err := s.sessionStore.Get(stream.Context(), request.SessionKey)
	if err != nil {
		return err
	}

	log := s.log.WithField("deviceId", session.GetDevice().GetId())
	log.Debug("incoming connection")

	trigger, err := s.devices.Add(session.GetDevice().GetId())
	if err != nil {
		// indicate that the client should retry
		return status.Error(codes.Unavailable, err.Error())
	}
	defer s.devices.Remove(session.GetDevice().GetId())

	metrics.DevicesConnected.Set(float64(s.devices.Length()))
	defer metrics.DevicesConnected.Set(float64(s.devices.Length()))

	if len(session.GetGroups()) == 0 {
		log.Warn("session with no groups detected")
	}

	timeout := time.After(time.Until(session.GetExpiry().AsTime()))
	updateDeviceTicker := time.NewTicker(1 * time.Minute)
	defer updateDeviceTicker.Stop()

	var lastCfg *pb.GetDeviceConfigurationResponse
	for {
		if cfg, err := s.makeDeviceConfiguration(stream.Context(), request.SessionKey); err != nil {
			log.WithError(err).Error("make device config")
		} else if equalDeviceConfigurations(lastCfg, cfg) {
			// no change, don't send
		} else {
			if err := stream.Send(cfg); err != nil {
				log.WithError(err).Debug("stream send for device failed")
			} else {
				lastCfg = cfg
			}
		}

		// block until trigger or done
		select {
		case <-trigger:
		case <-updateDeviceTicker.C:
		case <-stream.Context().Done():
			metrics.IncDeviceStreamsEnded("context_done")
			log.Debug("stream context done, tearing down")
			return nil
		case <-timeout:
			metrics.IncDeviceStreamsEnded("timeout")
			log.Debug("session timed out, tearing down")
			return nil
		}
	}
}

func equalDeviceConfigurations(a, b *pb.GetDeviceConfigurationResponse) bool {
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

	allGateways, err := s.db.ReadGateways(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user gateways: %w", err)
	}

	for i := range allGateways {
		allGateways[i].PasswordHash = ""
	}

	filters := []func(*pb.Gateway) bool{
		gatewayForUserGroups(session.GetGroups()),
	}

	// Hack to prevent windows users from connecting to the ms-login gateway
	if device.Platform == "windows" {
		filters = append(filters, not(gatewayHasName(GatewayMSLoginName)))
	}

	gateways := filterList(allGateways, filters...)

	metrics.DeviceConfigsReturned.WithLabelValues(device.Serial, device.Username).Inc()

	return &pb.GetDeviceConfigurationResponse{
		Status:   pb.DeviceConfigurationStatus_DeviceHealthy,
		Issues:   device.Issues,
		Gateways: gateways,
	}, nil
}

func (s *grpcServer) SendDeviceConfiguration(device *pb.Device) {
	s.devices.Trigger(device.GetId())
}

func (s *grpcServer) Login(ctx context.Context, r *pb.APIServerLoginRequest) (*pb.APIServerLoginResponse, error) {
	version := r.Version
	if version == "" {
		version = "unknown"
	}
	metrics.LoginRequests.WithLabelValues(version).Inc()

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
		s.log.WithError(err).Error("Error reading devices")
		return nil
	}
	/*
		device.ExternalID = fmt.Sprint(kolideDevice.ID)
		device.Issues = issuesByExternalID[device.ExternalID]

		if kolideDevice.LastSeenAt != nil {
			device.LastSeen = timestamppb.New(*kolideDevice.LastSeenAt)
		}
	*/

	kolideDevices, err := s.kolideClient.GetDevices(ctx)
	if err != nil {
		return fmt.Errorf("getting kolide devices: %w", err)
	}
	for _, kolideDevice := range kolideDevices {
		for _, device := range devices {
			if kolideDevice.Serial == device.Serial && kolideDevice.Platform == device.Platform {
				device.ExternalID = fmt.Sprint(kolideDevice.ID)
				if kolideDevice.LastSeenAt != nil {
					device.LastSeen = timestamppb.New(*kolideDevice.LastSeenAt)
				}
				break
			}
		}
	}

	issues, err := s.kolideClient.GetIssues(ctx)
	if err != nil {
		return fmt.Errorf("getting kolide issues: %w", err)
	}

	err = s.db.UpdateKolideIssues(ctx, issues)
	if err != nil {
		return fmt.Errorf("updating kolide issues: %w", err)
	}

	err = s.db.UpdateDevices(ctx, devices)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.log.WithError(err).Error("storing device")
	}

	// get fresh devices from db
	devices, err = s.db.ReadDevices(ctx)
	if err != nil {
		return fmt.Errorf("reading devices: %w", err)
	}

	for _, device := range devices {
		s.sessionStore.RefreshDevice(device)
		s.SendDeviceConfiguration(device)
	}

	return err
}
