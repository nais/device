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
	"github.com/nais/device/internal/device-agent/statemachine/state"
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

	cfg            *pb.GetDeviceConfigurationResponse
	connectedSince *timestamppb.Timestamp

	syncConfigLoop func(ctx context.Context) error
}

func New(
	rc runtimeconfig.RuntimeConfig,
	logger logrus.FieldLogger,
	notifier notify.Notifier,
	deviceHelper pb.DeviceHelperClient,
	statusUpdates chan<- *pb.AgentStatus,
) state.State {
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

func (c *Connected) Enter(ctx context.Context) state.EventWithSpan {
	// Set up WireGuard interface for communication with APIServer
	helperCtx, cancel := context.WithTimeout(ctx, helperTimeout)
	_, err := c.deviceHelper.Configure(helperCtx, c.rc.BuildHelperConfiguration([]*pb.Gateway{
		c.rc.APIServerPeer(),
	}))
	cancel()
	if err != nil {
		c.notifier.Errorf(err.Error())
		return state.SpanEvent(ctx, state.EventDisconnect)
	}

	// Teardown WireGuard interface when this state is finished
	defer func() {
		// need to base this context on background as `teardownCtx` is usually already cancelled when we get to this point.
		teardownCtx := trace.ContextWithSpan(context.Background(), trace.SpanFromContext(ctx))
		teardownCtx, cancel := context.WithTimeout(teardownCtx, helperTimeout)
		_, err = c.deviceHelper.Teardown(teardownCtx, &pb.TeardownRequest{})
		cancel()
		c.cfg = nil
	}()

	attempt := 0
	for ctx.Err() == nil {
		attempt++
		c.logger.WithField("attempt", attempt).Info("setting up gateway configuration stream...")
		err := c.syncConfigLoop(ctx)

		switch e := err; {
		case errors.Is(e, ErrUnavailable):
			c.logger.WithError(e).Warn("synchronize config: not connected to API server")
			time.Sleep(apiServerRetryInterval * time.Duration(math.Pow(float64(attempt), 3)))
			continue
		case errors.Is(e, auth.ErrTermsNotAccepted):
			c.notifier.Errorf("%v", e)
			return state.SpanEvent(ctx, state.EventDisconnect)
		case errors.Is(e, &auth.ParseTokenError{}):
			fallthrough
		case errors.Is(e, ErrUnauthenticated):
			c.notifier.Errorf("unauthenticated, please log in again.")
			c.rc.SetToken(nil)
			return state.SpanEvent(ctx, state.EventDisconnect)
		case errors.Is(e, io.EOF):
			c.logger.Info("connection unexpectedly lost (EOF), reconnecting...")
			attempt = 0
		case errors.Is(e, ErrLostConnection):
			c.logger.WithError(e).Info("lost connection, reconnecting...")
			attempt = 0
		case errors.Is(e, context.DeadlineExceeded):
			c.logger.WithError(e).Info("syncConfigLoop deadline exceeded")
			return state.SpanEvent(ctx, state.EventDisconnect)
		case errors.Is(e, context.Canceled):
			// in this case something from the outside canceled us, let them decide next state
			c.logger.WithError(e).Info("syncConfigLoop canceled")
			return state.SpanEvent(ctx, state.EventWaitForExternalEvent)
		case e != nil:
			// Unhandled error: disconnect
			c.logger.WithError(e).Error("error in syncConfigLoop")
			c.notifier.Errorf("Unhandled error while updating config. Please send your logs to the NAIS team")
			return state.SpanEvent(ctx, state.EventDisconnect)
		}
	}

	return state.SpanEvent(ctx, state.EventDisconnect)
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

	toInternalError := func(err error) error {
		switch grpcstatus.Code(err) {
		case codes.Unavailable:
			return ErrUnavailable
		case codes.Unauthenticated:
			return ErrUnauthenticated
		case codes.Canceled:
			return context.Canceled
		case codes.DeadlineExceeded:
			return context.DeadlineExceeded
		default:
			return err
		}
	}

	stream, cancel, err := c.syncSetup(ctx)
	if err != nil {
		return fmt.Errorf("setup gateway stream(%w): %w", toInternalError(err), err)
	}
	defer cancel()

	var healthCheckCancel context.CancelFunc = func() {}
	for ctx.Err() == nil {
		err := func() error {
			cfg, err := stream.Recv()
			healthCheckCancel()
			ctx, span := otel.Start(ctx, "SyncConfigLoop/recv")
			defer span.End()
			span.RecordError(err)

			if err != nil {
				internalErr := toInternalError(err)
				if internalErr == ErrUnavailable {
					// indicate that we had a working connection
					internalErr = ErrLostConnection
				}
				return fmt.Errorf("recv(%w): %w", internalErr, err)
			}

			c.logger.Info("received gateway configuration from API server")

			switch cfg.Status {
			case pb.DeviceConfigurationStatus_InvalidSession:
				span.AddEvent("session.invalid")
				return fmt.Errorf("invalid session (%w), config status: %v", ErrUnauthenticated, cfg.Status)
			case pb.DeviceConfigurationStatus_DeviceUnhealthy:
				span.AddEvent("device.unhealthy")

				c.logger.WithError(err).Error("device is not healthy")
				for _, issue := range cfg.Issues {
					c.logger.WithField("issue", issue).Error("issue detected")
				}
				if len(cfg.Issues) == 1 {
					c.notifier.Errorf("%v. Run '/msg @Kolide status' on Slack and fix the errors", cfg.Issues[0].Title)
				} else {
					// Make sure we do not report `Found 0 issues`
					count := ""
					if len(cfg.Issues) > 0 {
						count = fmt.Sprintf(" %v", len(cfg.Issues))
					}
					c.notifier.Errorf("Found%v issues on your device. Run '/msg @Kolide status' on Slack and fix the errors", count)
				}

				c.cfg = cfg

				c.triggerStatusUpdate()
				return nil
			case pb.DeviceConfigurationStatus_DeviceHealthy:
				span.AddEvent("device.healthy")

				c.logger.WithField("num_gateways", len(cfg.Gateways)).Info("device is healthy; got config")

				helperCtx, helperCancel := context.WithTimeout(ctx, helperTimeout)
				_, err = c.deviceHelper.Configure(helperCtx, c.rc.BuildHelperConfiguration(append(
					[]*pb.Gateway{
						c.rc.APIServerPeer(),
					},
					cfg.Gateways...,
				)))
				helperCancel()
				if err != nil {
					if ctx.Err() != nil {
						return err
					}

					return fmt.Errorf("configure helper(%w): %w", ErrUnavailable, err)
				}

				cfg.Gateways = pb.MergeGatewayHealth(c.cfg.GetGateways(), cfg.GetGateways())
				c.cfg = cfg

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

	location, err := time.LoadLocation("Europe/Oslo")
	if err == nil {
		c.connectedSince = timestamppb.New(time.Now().In(location))
	} else {
		c.connectedSince = timestamppb.Now()
	}
	c.logger.Info("gateway configuration stream established")

	return stream, func() {
		cancel()
		cleanup()
	}, nil
}

func (c Connected) String() string {
	return "Connected"
}

func (c Connected) Status() *pb.AgentStatus {
	state := pb.AgentState_Connected
	if c.cfg != nil && c.cfg.Status == pb.DeviceConfigurationStatus_DeviceUnhealthy {
		state = pb.AgentState_Unhealthy
	}

	return &pb.AgentStatus{
		ConnectedSince:  c.connectedSince,
		Gateways:        c.cfg.GetGateways(),
		Issues:          c.cfg.GetIssues(),
		ConnectionState: state,
	}
}

func (c *Connected) launchHealthCheck(ctx context.Context) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	gateways := c.cfg.GetGateways()

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
				c.logger.WithField("num_gateways", total).Debug("pinging gateways...")
				for i, gw := range gateways {
					wg.Add(1)
					go func(i int, gw *pb.Gateway) {
						defer wg.Done()
						err := ping(c.logger, gw.Ipv4)
						pos := fmt.Sprintf("[%02d/%02d]", i+1, total)
						if err == nil {
							gw.Healthy = true
							c.logger.WithFields(logrus.Fields{
								"num":     pos,
								"gateway": gw.Name,
								"ip":      gw.Ipv4,
							}).Debug("successfully pinged")
						} else {
							gw.Healthy = false
							c.logger.WithError(err).WithFields(logrus.Fields{
								"num":     pos,
								"gateway": gw.Name,
								"ip":      gw.Ipv4,
							}).Debug("unable to ping")
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
		log.WithError(err).Error("closing ping connection")
	}

	return nil
}
