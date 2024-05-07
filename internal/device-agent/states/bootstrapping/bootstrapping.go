package bootstrapping

import (
	"context"
	"time"

	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
)

type Bootstrapping struct {
	rc           runtimeconfig.RuntimeConfig
	logger       logrus.FieldLogger
	notifier     notify.Notifier
	deviceHelper pb.DeviceHelperClient
}

func New(rc runtimeconfig.RuntimeConfig, logger logrus.FieldLogger, notifier notify.Notifier, deviceHelper pb.DeviceHelperClient) statemachine.State {
	return &Bootstrapping{
		rc:           rc,
		notifier:     notifier,
		deviceHelper: deviceHelper,
		logger:       logger,
	}
}

func (b *Bootstrapping) Enter(ctx context.Context) statemachine.Event {
	ctx, span := otel.Start(ctx, "Bootstrapping")
	defer span.End()

	if err := b.rc.LoadEnrollConfig(); err == nil {
		span.AddEvent("enroll.loaded")
		b.logger.Infof("Loaded enroll")
	} else {
		span.AddEvent("enroll.new")
		b.logger.Infof("Unable to load enroll config: %s", err)
		b.logger.Infof("Enrolling device")
		enrollCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		serial, err := b.deviceHelper.GetSerial(enrollCtx, &pb.GetSerialRequest{})
		if err != nil {
			span.RecordError(err)
			b.notifier.Errorf("Unable to get serial number: %v", err)
			return statemachine.EventDisconnect
		}

		err = b.rc.EnsureEnrolled(enrollCtx, serial.GetSerial())

		cancel()
		if err != nil {
			span.RecordError(err)
			b.notifier.Errorf("Bootstrap: %v", err)
			return statemachine.EventDisconnect
		}
	}

	return statemachine.EventBootstrapped
}

func (Bootstrapping) AgentState() pb.AgentState {
	return pb.AgentState_Bootstrapping
}

func (b Bootstrapping) String() string {
	return "Bootstrapping"
}

func (b Bootstrapping) Status() *pb.AgentStatus {
	return &pb.AgentStatus{
		ConnectionState: b.AgentState(),
	}
}
