package connected

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/version"
)

const (
	apiServerRetryInterval = time.Millisecond * 10
	healthCheckInterval    = 20 * time.Second // how often to healthcheck gateways
	helperTimeout          = 20 * time.Second
)

type Connected struct {
	rc            runtimeconfig.RuntimeConfig
	logger        logrus.FieldLogger
	notifier      notify.Notifier
	deviceHelper  pb.DeviceHelperClient
	statusUpdates chan<- *pb.AgentStatus

	gateways       []*pb.Gateway
	connectedSince *timestamppb.Timestamp
	unhealthy      bool

	syncConfigLoop func(ctx context.Context) error
}

func New(
	rc runtimeconfig.RuntimeConfig,
	logger logrus.FieldLogger,
	notifier notify.Notifier,
	deviceHelper pb.DeviceHelperClient,
	statusUpdates chan<- *pb.AgentStatus,
) statemachine.State {
	c := &Connected{
		rc:            rc,
		logger:        logger,
		notifier:      notifier,
		deviceHelper:  deviceHelper,
		statusUpdates: statusUpdates,
	}
	c.syncConfigLoop = c.defaultSyncConfigLoop
	return c
}

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrUnavailable     = errors.New("unavailable")
	ErrLostConnection  = errors.New("lost connection")
)

func (c *Connected) Enter(ctx context.Context) statemachine.EventWithSpan {
	// Set up WireGuard interface for communication with APIServer
	helperCtx, cancel := context.WithTimeout(ctx, helperTimeout)
	_, err := c.deviceHelper.Configure(helperCtx, c.rc.BuildHelperConfiguration([]*pb.Gateway{
		c.rc.APIServerPeer(),
	}))
	cancel()
	if err != nil {
		c.notifier.Errorf(err.Error())
		return statemachine.SpanEvent(ctx, statemachine.EventDisconnect)
	}

	// Teardown WireGuard interface when this state is finished
	defer func() {
		// need to base this context on background as `ctx` is usually already cancelled when we get to this point.
		ctx, cancel := context.WithTimeout(context.Background(), helperTimeout)
		_, err = c.deviceHelper.Teardown(ctx, &pb.TeardownRequest{})
		cancel()
		c.gateways = nil
		c.unhealthy = false
	}()

	attempt := 0
	for ctx.Err() == nil {
		attempt++
		c.logger.Infof("[attempt %d] Setting up gateway configuration stream...", attempt)
		err := c.syncConfigLoop(ctx)

		switch e := err; {
		case errors.Is(e, ErrUnavailable):
			c.logger.Warnf("Synchronize config: not connected to API server: %v", err)
			time.Sleep(apiServerRetryInterval * time.Duration(math.Pow(float64(attempt), 3)))
			continue
		case errors.Is(e, auth.ErrTermsNotAccepted):
			c.notifier.Errorf("%v", e)
			return statemachine.SpanEvent(ctx, statemachine.EventDisconnect)
		case errors.Is(e, &auth.ParseTokenError{}):
			fallthrough
		case errors.Is(e, ErrUnauthenticated):
			c.notifier.Errorf("Unauthenticated: %v", err)
			c.rc.SetToken(nil)
			return statemachine.SpanEvent(ctx, statemachine.EventDisconnect)
		case errors.Is(e, io.EOF):
			c.logger.Infof("Connection unexpectedly lost (EOF), reconnecting...")
			attempt = 0
		case errors.Is(e, ErrLostConnection):
			c.logger.Infof("Lost connection, reconnecting...")
			attempt = 0
		case errors.Is(e, context.DeadlineExceeded):
			c.logger.Infof("syncConfigLoop deadline exceeded: %v", err)
			return statemachine.SpanEvent(ctx, statemachine.EventDisconnect)
		case errors.Is(e, context.Canceled):
			// in this case something from the outside canceled us, let them decide next state
			c.logger.Infof("syncConfigLoop canceled: %v", err)
			return statemachine.SpanEvent(ctx, statemachine.EventWaitForExternalEvent)
		case e != nil:
			// Unhandled error: disconnect
			c.logger.Errorf("error in syncConfigLoop: %v", err)
			c.notifier.Errorf("Unhandled error while updating config. Please send your logs to the NAIS team.")
			return statemachine.SpanEvent(ctx, statemachine.EventDisconnect)
		}
	}

	return statemachine.SpanEvent(ctx, statemachine.EventDisconnect)
}

func (c *Connected) triggerStatusUpdate() {
	select {
	case c.statusUpdates <- c.Status():
	default:
	}
}

func (c *Connected) login(ctx context.Context, apiserverClient pb.APIServerClient, session *pb.Session) (*pb.Session, error) {
	if !session.Expired() {
		return session, nil
	}

	ctx, span := otel.Start(ctx, "Login")
	defer span.End()

	serial, err := c.deviceHelper.GetSerial(ctx, &pb.GetSerialRequest{})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	token, err := c.rc.GetToken(ctx)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	loginResponse, err := apiserverClient.Login(ctx, &pb.APIServerLoginRequest{
		Token:    token,
		Platform: config.Platform,
		Serial:   serial.GetSerial(),
		Version:  version.Version,
	})
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	if err := c.rc.SetTenantSession(loginResponse.Session); err != nil {
		span.RecordError(err)
		return nil, err
	}

	return loginResponse.Session, nil
}

func (c *Connected) defaultSyncConfigLoop(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	stream, cancel, err := c.syncSetup(ctx)
	if err != nil {
		return err
	}
	defer cancel()

	var healthCheckCancel context.CancelFunc = func() {}
	for ctx.Err() == nil {
		err := func() error {
			cfg, err := stream.Recv()
			healthCheckCancel()
			ctx, span := otel.Start(ctx, "SyncConfigLoop/recv", trace.WithNewRoot())
			defer span.End()
			span.RecordError(err)

			switch e := err; {
			case grpcstatus.Code(e) == codes.Unavailable:
				return fmt.Errorf("recv(%w): %w", ErrLostConnection, e)
			case grpcstatus.Code(e) == codes.Unauthenticated:
				return fmt.Errorf("recv(%w): %w", ErrUnauthenticated, e)
			case grpcstatus.Code(e) == codes.Canceled:
				return fmt.Errorf("recv(%w): %w", context.Canceled, e)
			case grpcstatus.Code(e) == codes.DeadlineExceeded:
				return fmt.Errorf("recv(%w): %w", context.DeadlineExceeded, e)
			case e != nil:
				return fmt.Errorf("recv: %w", e)
			}

			c.logger.Infof("Received gateway configuration from API server")

			switch cfg.Status {
			case pb.DeviceConfigurationStatus_InvalidSession:
				span.AddEvent("session.invalid")
				return fmt.Errorf("invalid session (%w), config status: %v", ErrUnauthenticated, cfg.Status)
			case pb.DeviceConfigurationStatus_DeviceUnhealthy:
				span.AddEvent("device.unhealthy")

				c.logger.Errorf("Device is not healthy: %v", err)
				c.notifier.Errorf("No access as your device is unhealthy. Run '/msg @Kolide status' on Slack and fix the errors")

				c.unhealthy = true
				c.gateways = nil

				c.triggerStatusUpdate()
				return nil
			case pb.DeviceConfigurationStatus_DeviceHealthy:
				span.AddEvent("device.healthy")

				c.logger.Infof("Device is healthy; server pushed %d gateways", len(cfg.Gateways))

				c.unhealthy = false

				helperCtx, helperCancel := context.WithTimeout(ctx, helperTimeout)
				_, err = c.deviceHelper.Configure(helperCtx, c.rc.BuildHelperConfiguration(append(
					[]*pb.Gateway{
						c.rc.APIServerPeer(),
					},
					cfg.Gateways...,
				)))
				helperCancel()
				if err != nil {
					return fmt.Errorf("configure helper: %w", err)
				}

				c.gateways = pb.MergeGatewayHealth(c.gateways, cfg.Gateways)
				c.triggerStatusUpdate()
				healthCheckCancel = c.launchHealthCheck(ctx)
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	return ctx.Err()
}

func (c *Connected) syncSetup(ctx context.Context) (pb.APIServer_GetDeviceConfigurationClient, context.CancelFunc, error) {
	ctx, span := otel.Start(ctx, "SyncConfigLoop/setup")
	defer span.End()

	session, err := c.rc.GetTenantSession()
	if err != nil {
		return nil, nil, err
	}

	apiserverClient, cleanup, err := c.rc.ConnectToAPIServer(ctx)
	if err != nil {
		if grpcstatus.Code(err) == codes.Unavailable {
			return nil, nil, fmt.Errorf("connect to apiserver(%w): %w", ErrUnavailable, err)
		}
		return nil, nil, err
	}

	session, err = c.login(ctx, apiserverClient, session)
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	streamContext, cancel := context.WithDeadline(ctx, session.Expiry.AsTime())

	stream, err := apiserverClient.GetDeviceConfiguration(streamContext, &pb.GetDeviceConfigurationRequest{
		SessionKey: session.Key,
	})
	if err != nil {
		cancel()
		cleanup()
		return nil, nil, err
	}

	c.connectedSince = timestamppb.Now()
	c.logger.Infof("Gateway configuration stream established")

	return stream, func() {
		cancel()
		cleanup()
	}, nil
}

func (c Connected) AgentState() pb.AgentState {
	if c.unhealthy {
		return pb.AgentState_Unhealthy
	} else {
		return pb.AgentState_Connected
	}
}

func (c Connected) String() string {
	return "Connected"
}

func (c Connected) Status() *pb.AgentStatus {
	return &pb.AgentStatus{
		ConnectedSince:  c.connectedSince,
		Gateways:        c.gateways,
		ConnectionState: c.AgentState(),
	}
}

func (c *Connected) launchHealthCheck(ctx context.Context) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	gateways := c.gateways

	go func() {
		timer := time.NewTimer(10 * time.Millisecond)
		for ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				timer.Reset(healthCheckInterval)
				wg := &sync.WaitGroup{}

				total := len(gateways)
				c.logger.Debugf("Pinging %d gateways...", total)
				for i, gw := range gateways {
					wg.Add(1)
					go func(i int, gw *pb.Gateway) {
						defer wg.Done()
						err := ping(c.logger, gw.Ipv4)
						pos := fmt.Sprintf("[%02d/%02d]", i+1, total)
						if err == nil {
							gw.Healthy = true
							c.logger.Debugf("%s %s: successfully pinged %v", pos, gw.Name, gw.Ipv4)
						} else {
							gw.Healthy = false
							c.logger.Debugf("%s %s: unable to ping %s: %v", pos, gw.Name, gw.Ipv4, err)
						}
					}(i, gw)
				}
				wg.Wait()
				c.triggerStatusUpdate()
			}
		}
	}()

	return cancel
}

func ping(log logrus.FieldLogger, addr string) error {
	c, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", addr, "3000"), 2*time.Second)
	if err != nil {
		return err
	}

	err = c.Close()
	if err != nil {
		log.Errorf("closing ping connection: %v", err)
	}

	return nil
}
