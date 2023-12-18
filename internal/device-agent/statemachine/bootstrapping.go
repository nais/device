package statemachine

import (
	"context"
	"fmt"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/version"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"math"
	"time"
)

const (
	apiServerRetryInterval = time.Millisecond * 10
	helperTimeout          = 20 * time.Second
	syncConfigDialTimeout  = 1 * time.Second // sleep time between failed configuration syncs
)

type Bootstrapping struct {
	rc           runtimeconfig.RuntimeConfig
	cfg          config.Config
	notifier     notify.Notifier
	deviceHelper pb.DeviceHelperClient
	logger       logrus.FieldLogger
}

func (b *Bootstrapping) Enter(ctx context.Context, sendEvent func(Event)) {
	if err := b.rc.LoadEnrollConfig(); err == nil {
		b.logger.Infof("Loaded enroll")
	} else {
		b.logger.Infof("Unable to load enroll config: %s", err)
		b.logger.Infof("Enrolling device")
		enrollCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		serial, err := b.deviceHelper.GetSerial(enrollCtx, &pb.GetSerialRequest{})
		if err != nil {
			b.notifier.Errorf("Unable to get serial number: %v", err)
			sendEvent(EventDisconnect)
			cancel()
			return
		}

		err = b.rc.EnsureEnrolled(enrollCtx, serial.GetSerial())

		cancel()
		if err != nil {
			b.notifier.Errorf("Bootstrap: %v", err)
			sendEvent(EventDisconnect)
			return
		}
	}

	helperCtx, cancel := context.WithTimeout(ctx, helperTimeout)
	_, err := b.deviceHelper.Configure(helperCtx, b.rc.BuildHelperConfiguration([]*pb.Gateway{
		b.rc.APIServerPeer(),
	}))
	cancel()

	if err != nil {
		b.notifier.Errorf(err.Error())
		sendEvent(EventDisconnect)
		return
	}

	// TODO: status.ConnectedSince = timestamppb.Now()

	gateways := make(chan []*pb.Gateway, 16) // TODO: Do something with this
	attempt := 0
	for ctx.Err() == nil {
		attempt++
		b.logger.Infof("[attempt %d] Setting up gateway configuration stream...", attempt)
		err := b.syncConfigLoop(ctx, gateways, sendEvent)

		switch grpcstatus.Code(err) {
		case codes.OK:
			attempt = 0
		case codes.Unavailable:
			b.logger.Warnf("Synchronize config: not connected to API server: %v", err)
			time.Sleep(apiServerRetryInterval * time.Duration(math.Pow(float64(attempt), 3)))
		case codes.Unauthenticated:
			b.logger.Errorf("Logging in: %s", err)
			b.rc.SetToken(nil)
			b.logger.Error("Cleaned up old tokens")
			sendEvent(EventDisconnect)
			return
		default:
			b.notifier.Errorf(err.Error())
			b.logger.Errorf("error in syncConfigLoop: %v", err)
			b.logger.Errorf("Assuming invalid session; disconnecting.")
			sendEvent(EventDisconnect)
			return
		}
	}

	b.logger.Infof("Gateway config synchronization loop: %s", ctx.Err())
}

func (b *Bootstrapping) syncConfigLoop(ctx context.Context, gateways chan<- []*pb.Gateway, sendEvent func(Event)) error {
	dialContext, cancel := context.WithTimeout(ctx, syncConfigDialTimeout)
	defer cancel()

	conn, err := b.rc.DialAPIServer(dialContext)
	if err != nil {
		return grpcstatus.Errorf(codes.Unavailable, err.Error())
	}
	b.logger.Infof("Connected to API server")

	defer conn.Close()

	apiserverClient := pb.NewAPIServerClient(conn) // TODO: Make this testable

	session, err := b.rc.GetTenantSession()
	if err != nil {
		return err
	}

	if session.Expired() {
		serial, err := b.deviceHelper.GetSerial(ctx, &pb.GetSerialRequest{})
		if err != nil {
			return err
		}

		token, err := b.rc.GetToken(ctx)
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

		if err := b.rc.SetTenantSession(loginResponse.Session); err != nil {
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

	b.logger.Infof("Gateway configuration stream established")

	// TODO: Already?
	sendEvent(EventBootstrapped)

	for ctx.Err() == nil {
		cfg, err := stream.Recv()
		if err != nil {
			return err
		}

		b.logger.Infof("Received gateway configuration from API server")

		switch cfg.Status {
		case pb.DeviceConfigurationStatus_InvalidSession:
			b.logger.Errorf("Unauthorized access from apiserver: %v", err)
			return fmt.Errorf("config status: %v", cfg.Status)
		case pb.DeviceConfigurationStatus_DeviceUnhealthy:
			b.logger.Errorf("Device is not healthy: %v", err)
			b.notifier.Errorf("No access as your device is unhealthy. Run '/msg @Kolide status' on Slack and fix the errors")
			// TODO: Need new event? das.stateChange <- pb.AgentState_Unhealthy
			continue
		case pb.DeviceConfigurationStatus_DeviceHealthy:
			b.logger.Infof("Device is healthy; server pushed %d gateways", len(cfg.Gateways))
		default:
		}

		gateways <- cfg.Gateways
	}

	return nil
}
