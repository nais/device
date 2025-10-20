package bootstrapping

import (
	"context"
	"time"

	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine/state"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
)

type Bootstrapping struct {
	rc           runtimeconfig.RuntimeConfig
	logger       logrus.FieldLogger
	notifier     notify.Notifier
	deviceHelper pb.DeviceHelperClient
}

func New(rc runtimeconfig.RuntimeConfig, logger logrus.FieldLogger, notifier notify.Notifier, deviceHelper pb.DeviceHelperClient) state.State {
	return &Bootstrapping{
		rc:           rc,
		notifier:     notifier,
		deviceHelper: deviceHelper,
		logger:       logger,
	}
}

func (b *Bootstrapping) Enter(ctx context.Context) state.EventWithSpan {
	ctx, span := otel.Start(ctx, "Bootstrapping")
	defer span.End()

	if err := b.rc.LoadEnrollConfig(); err == nil {
		span.AddEvent("enroll.loaded")
		b.logger.Info("loaded enroll")
		return state.SpanEvent(ctx, state.EventBootstrapped)
	}

	span.AddEvent("enroll.new")
	b.logger.Info("enrolling device")
	enrollCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	serial, err := b.deviceHelper.GetSerial(enrollCtx, &pb.GetSerialRequest{})
	if err != nil {
		span.RecordError(err)
		b.notifier.Errorf("Unable to get serial number: %v", err)
		return state.SpanEvent(ctx, state.EventDisconnect)
	}

	err = b.rc.EnsureEnrolled(enrollCtx, serial.GetSerial())

	cancel()
	if err != nil {
		span.RecordError(err)
		b.notifier.Errorf("Bootstrap: %v", err)
		return state.SpanEvent(ctx, state.EventDisconnect)
	}

	return state.SpanEvent(ctx, state.EventBootstrapped)
}

func (b Bootstrapping) String() string {
	return "Bootstrapping"
}

func (b Bootstrapping) Status() *pb.AgentStatus {
	return &pb.AgentStatus{
		ConnectionState: pb.AgentState_Bootstrapping,
	}
}
