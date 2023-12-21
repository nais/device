package statemachine

import (
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"math"
	"net"
	"sync"
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
	healthCheckInterval    = 20 * time.Second // how often to healthcheck gateways
)

type Connected struct {
	baseState
	deviceHelper        pb.DeviceHelperClient
	triggerStatusUpdate func()
	gateways            []*pb.Gateway
	connectedSince      *timestamppb.Timestamp
	unhealthy           bool
}

func (c *Connected) Enter(ctx context.Context) Event {
	// Set up WireGuard interface for communication with APIServer
	helperCtx, cancel := context.WithTimeout(ctx, helperTimeout)
	_, err := c.deviceHelper.Configure(helperCtx, c.rc.BuildHelperConfiguration([]*pb.Gateway{
		c.rc.APIServerPeer(),
	}))
	cancel()
	if err != nil {
		c.notifier.Errorf(err.Error())
		return EventDisconnect
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

		switch grpcstatus.Code(err) {
		case codes.OK:
			attempt = 0
			continue
		case codes.Unavailable:
			c.logger.Warnf("Synchronize config: not connected to API server: %v", err)
			time.Sleep(apiServerRetryInterval * time.Duration(math.Pow(float64(attempt), 3)))
			continue
		case codes.Unauthenticated:
			c.notifier.Errorf("Unauthenticated: %v", err)
			c.rc.SetToken(nil)
			return EventDisconnect
		}

		if errors.Is(ctx.Err(), context.Canceled) {
			// we got cancelled (most likely shutdown or disconnect event)
			break
		}

		if err != nil {
			// Unhandled error: disconnect
			c.logger.Errorf("error in syncConfigLoop: %v", err)
			c.notifier.Errorf("Unhandled error while updating config. Plase send your logs to the NAIS team.")
			return EventDisconnect
		}
	}

	c.logger.Infof("Config sync loop done: %s", ctx.Err())
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

	var healthCheckCancel context.CancelFunc = func() {}

	for ctx.Err() == nil {
		cfg, err := stream.Recv()
		healthCheckCancel()
		if err != nil {
			return fmt.Errorf("recv: %w", err)
		}

		c.logger.Infof("Received gateway configuration from API server")

		switch cfg.Status {
		case pb.DeviceConfigurationStatus_InvalidSession:
			return grpcstatus.Errorf(codes.Unauthenticated, "invalid session, config status: %v", cfg.Status)
		case pb.DeviceConfigurationStatus_DeviceUnhealthy:
			c.logger.Errorf("Device is not healthy: %v", err)
			c.notifier.Errorf("No access as your device is unhealthy. Run '/msg @Kolide status' on Slack and fix the errors")

			c.unhealthy = true
			c.gateways = nil

			c.triggerStatusUpdate()
			continue
		case pb.DeviceConfigurationStatus_DeviceHealthy:
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
	}

	return ctx.Err()
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
		Tenants:         c.baseStatus.GetTenants(),
		Gateways:        c.gateways,
		ConnectionState: c.AgentState(),
	}
}

func (c *Connected) launchHealthCheck(ctx context.Context) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	gateways := c.gateways

	go func() {
		timer := time.NewTimer(time.Millisecond)
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
						err := ping(c.logger, gw.Ipv4)
						pos := fmt.Sprintf("[%02d/%02d]", i+1, total)
						if err == nil {
							gw.Healthy = true
							c.logger.Debugf("%s %s: successfully pinged %v", pos, gw.Name, gw.Ipv4)
						} else {
							gw.Healthy = false
							c.logger.Debugf("%s %s: unable to ping %s: %v", pos, gw.Name, gw.Ipv4, err)
						}
						wg.Done()
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
