package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nais/device/internal/apiserver/metrics"
	"github.com/nais/device/pkg/pb"
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

	device, err := s.db.ReadDeviceByID(ctx, session.GetDevice().GetId())
	if err != nil {
		return nil, err
	}

	var sessionIssues []*pb.DeviceIssue
	if s.kolideEnabled {
		if acceptedAt, err := s.db.GetAcceptedAt(ctx, session.ObjectID); err != nil {
			return nil, err
		} else if acceptedAt == nil {
			now := timestamppb.Now()
			sessionIssues = append(sessionIssues, &pb.DeviceIssue{
				Title:         "Do's and don'ts not accepted",
				Message:       "In order to use naisdevice you have to accept our Do's and don'ts. Click on naisdevice and then the Acceptable use policy menu item to accept.",
				Severity:      pb.Severity_Critical,
				DetectedAt:    now,
				LastUpdated:   now,
				ResolveBefore: now,
			})
		}
	}

	if !device.Healthy() || len(sessionIssues) > 0 {
		return &pb.GetDeviceConfigurationResponse{
			Status: pb.DeviceConfigurationStatus_DeviceUnhealthy,
			Issues: append(device.Issues, sessionIssues...),
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

	kolideDevices, err := s.kolideClient.GetDevices(ctx)
	if err != nil {
		return fmt.Errorf("getting kolide devices: %w", err)
	}

	// Clear all external_ids first - they will be set again if a match is found
	for _, device := range devices {
		device.ExternalID = ""
	}

	// Set external_id for devices that match Kolide devices
	for _, kolideDevice := range kolideDevices {
		for _, device := range devices {
			if kolideDevice.Serial == device.Serial && kolideDevice.Platform == device.Platform {
				device.ExternalID = kolideDevice.ID
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

func (s *grpcServer) GetAcceptableUseAcceptedAt(ctx context.Context, req *pb.GetAcceptableUseAcceptedAtRequest) (*pb.GetAcceptableUseAcceptedAtResponse, error) {
	session, err := s.sessionStore.Get(ctx, req.GetSessionKey())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unknown session")
	}

	if session.Expired() {
		return nil, status.Error(codes.Unauthenticated, "session expired")
	}

	if acceptedAt, err := s.db.GetAcceptedAt(ctx, session.GetObjectID()); err != nil {
		return nil, status.Errorf(codes.Internal, "unable to get acceptance: %v", err)
	} else {
		return &pb.GetAcceptableUseAcceptedAtResponse{AcceptedAt: acceptedAt}, nil
	}
}

func (s *grpcServer) SetAcceptableUseAccepted(ctx context.Context, req *pb.SetAcceptableUseAcceptedRequest) (*pb.SetAcceptableUseAcceptedResponse, error) {
	session, err := s.sessionStore.Get(ctx, req.GetSessionKey())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unknown session")
	}

	if session.Expired() {
		return nil, status.Error(codes.Unauthenticated, "session expired")
	}

	if req.Accepted {
		return &pb.SetAcceptableUseAcceptedResponse{}, s.db.AcceptAcceptableUse(ctx, session.GetObjectID())
	}

	return &pb.SetAcceptableUseAcceptedResponse{}, s.db.RejectAcceptableUse(ctx, session.GetObjectID())
}

func (s *grpcServer) GetGatewayJitaGrantsForUser(ctx context.Context, req *pb.GetGatewayJitaGrantsForUserRequest) (*pb.GetGatewayJitaGrantsForUserResponse, error) {
	session, err := s.sessionStore.Get(ctx, req.GetSessionKey())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unknown session")
	}

	if session.Expired() {
		return nil, status.Error(codes.Unauthenticated, "session expired")
	}

	grants, err := s.db.GetGatewayJitaGrantsForUser(ctx, session.GetObjectID())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to get gateway jita grants: %v", err)
	}

	return &pb.GetGatewayJitaGrantsForUserResponse{
		GatewayJitaGrants: grants,
	}, nil
}

func (s *grpcServer) UserHasAccessToPrivilegedGateway(ctx context.Context, req *pb.UserHasAccessToPrivilegedGatewayRequest) (*pb.UserHasAccessToPrivilegedGatewayResponse, error) {
	session, err := s.sessionStore.Get(ctx, req.GetSessionKey())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unknown session")
	}

	if session.Expired() {
		return nil, status.Error(codes.Unauthenticated, "session expired")
	}

	hasAccess, err := s.db.UserHasAccessToPrivilegedGateway(ctx, session.GetObjectID(), req.Gateway)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to query database: %v", err)
	}

	return &pb.UserHasAccessToPrivilegedGatewayResponse{
		HasAccess: hasAccess,
	}, nil
}

func (s *grpcServer) GrantPrivilegedGatewayAccess(ctx context.Context, req *pb.GrantPrivilegedGatewayAccessRequest) (*pb.GrantPrivilegedGatewayAccessResponse, error) {
	session, err := s.sessionStore.Get(ctx, req.GetSessionKey())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unknown session")
	}

	if err := s.authenticator.ValidateJita(session, req.Token); err != nil {
		s.log.WithError(err).Error("validate token")
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	if session.Expired() {
		return nil, status.Error(codes.Unauthenticated, "session expired")
	}

	n := req.GetNewPrivilegedGatewayAccess()
	if n == nil {
		return nil, status.Error(codes.InvalidArgument, "no new privileged gateway access")
	}

	if err := s.db.GrantPrivilegedGatewayAccess(ctx, session.GetObjectID(), n.Gateway, n.Expires.AsTime(), n.Reason); err != nil {
		return nil, status.Errorf(codes.Internal, "unable to grant privileged gateway access: %v", err)
	}

	s.gateways.Trigger(req.NewPrivilegedGatewayAccess.Gateway)

	return &pb.GrantPrivilegedGatewayAccessResponse{}, nil
}

func (s *grpcServer) RevokePrivilegedGatewayAccess(ctx context.Context, req *pb.RevokePrivilegedGatewayAccessRequest) (*pb.RevokePrivilegedGatewayAccessResponse, error) {
	session, err := s.sessionStore.Get(ctx, req.GetSessionKey())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unknown session")
	}

	if session.Expired() {
		return nil, status.Error(codes.Unauthenticated, "session expired")
	}

	if err := s.db.RevokePrivilegedGatewayAccess(ctx, session.GetObjectID(), req.Gateway); err != nil {
		return nil, status.Errorf(codes.Internal, "unable to revoke privileged gateway access: %v", err)
	}

	s.gateways.Trigger(req.Gateway)

	return &pb.RevokePrivilegedGatewayAccessResponse{}, nil
}
