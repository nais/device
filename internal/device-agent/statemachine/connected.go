package statemachine

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/version"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	apiServerRetryInterval = time.Millisecond * 10
)

type Connected struct {
	BaseState
	deviceHelper        pb.DeviceHelperClient
	triggerStatusUpdate func()
	gateways            []*pb.Gateway
	connectedSince      *timestamppb.Timestamp
}

func (c *Connected) Enter(ctx context.Context) Event {
	attempt := 0
	for ctx.Err() == nil {
		attempt++
		c.logger.Infof("[attempt %d] Setting up gateway configuration stream...", attempt)
		err := c.syncConfigLoop(ctx)

		switch grpcstatus.Code(err) {
		case codes.OK:
			attempt = 0
		case codes.Unavailable:
			c.logger.Warnf("Synchronize config: not connected to API server: %v", err)
			time.Sleep(apiServerRetryInterval * time.Duration(math.Pow(float64(attempt), 3)))
		case codes.Unauthenticated:
			c.logger.Errorf("Logging in: %s", err)
			c.rc.SetToken(nil)
			c.logger.Error("Cleaned up old tokens")
			return EventDisconnect
		default:
			// TODO: More granular error handling
			c.notifier.Errorf(err.Error())
			c.logger.Errorf("error in syncConfigLoop: %v", err)
			c.logger.Errorf("Assuming invalid session; disconnecting.")
			return EventDisconnect
		}
	}

	c.logger.Infof("Gateway config synchronization loop: %s", ctx.Err())
	return EventWaitForExternalEvent
}

func (c *Connected) syncConfigLoop(ctx context.Context) error {
	apiserverClient, cleanup, err := c.rc.ConnectToAPIServer(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	session, err := c.rc.GetTenantSession()
	if err != nil {
		return err
	}

	if session.Expired() {
		serial, err := c.deviceHelper.GetSerial(ctx, &pb.GetSerialRequest{})
		if err != nil {
			return err
		}

		token, err := c.rc.GetToken(ctx)
		if err != nil {
			return err
		}

		loginResponse, err := apiserverClient.Login(ctx, &pb.APIServerLoginRequest{
			Token:    token,
			Platform: config.Platform,
			Serial:   serial.GetSerial(),
			Version:  version.Version,
		})
		if err != nil {
			return err
		}

		if err := c.rc.SetTenantSession(loginResponse.Session); err != nil {
			return err
		}
		session = loginResponse.Session
	}

	streamContext, cancel := context.WithDeadline(ctx, session.Expiry.AsTime())
	defer cancel()

	stream, err := apiserverClient.GetDeviceConfiguration(streamContext, &pb.GetDeviceConfigurationRequest{
		SessionKey: session.Key,
	})
	if err != nil {
		return err
	}

	c.connectedSince = timestamppb.Now()
	c.logger.Infof("Gateway configuration stream established")

	for ctx.Err() == nil {
		cfg, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("recv: %w", err)
		}

		c.logger.Infof("Received gateway configuration from API server")

		switch cfg.Status {
		case pb.DeviceConfigurationStatus_InvalidSession:
			c.logger.Errorf("Unauthorized access from apiserver: %v", err)
			return fmt.Errorf("config status: %v", cfg.Status)
		case pb.DeviceConfigurationStatus_DeviceUnhealthy:
			c.logger.Errorf("Device is not healthy: %v", err)
			c.notifier.Errorf("No access as your device is unhealthy. Run '/msg @Kolide status' on Slack and fix the errors")
			// TODO: Need new event? das.stateChange <- pb.AgentState_Unhealthy
			continue
		case pb.DeviceConfigurationStatus_DeviceHealthy:
			c.logger.Infof("Device is healthy; server pushed %d gateways", len(cfg.Gateways))
		default:
		}

		helperCtx, helperCancel := context.WithTimeout(ctx, helperTimeout)
		_, err = c.deviceHelper.Configure(helperCtx, c.rc.BuildHelperConfiguration(append(
			[]*pb.Gateway{
				c.rc.APIServerPeer(),
			},
			cfg.Gateways...,
		)))
		helperCancel()
		pb.MergeGatewayHealth(c.gateways, cfg.Gateways)
		c.triggerStatusUpdate()
	}

	return nil
}

func (Connected) AgentState() pb.AgentState {
	return pb.AgentState_Connected
}

func (c Connected) String() string {
	return c.AgentState().String()
}

func (c Connected) Status() *pb.AgentStatus {
	return &pb.AgentStatus{
		ConnectedSince:  c.connectedSince,
		Tenants:         c.baseStatus.GetTenants(),
		Gateways:        c.gateways,
		ConnectionState: c.AgentState(),
	}
}
