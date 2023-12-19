package statemachine

import (
	"context"
	"time"

	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
)

const (
	helperTimeout = 20 * time.Second
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

	sendEvent(EventBootstrapped)
	// TODO: status.ConnectedSince = timestamppb.Now()
}

func (Bootstrapping) AgentState() pb.AgentState {
	return pb.AgentState_Bootstrapping
}

func (b Bootstrapping) String() string {
	return b.AgentState().String()
}
