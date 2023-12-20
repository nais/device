package statemachine

import (
	"context"
	"time"

	"github.com/nais/device/internal/pb"
)

const (
	helperTimeout = 20 * time.Second
)

type Bootstrapping struct {
	baseState
	deviceHelper pb.DeviceHelperClient
}

func (b *Bootstrapping) Enter(ctx context.Context) Event {
	if err := b.rc.LoadEnrollConfig(); err == nil {
		b.logger.Infof("Loaded enroll")
	} else {
		b.logger.Infof("Unable to load enroll config: %s", err)
		b.logger.Infof("Enrolling device")
		enrollCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		serial, err := b.deviceHelper.GetSerial(enrollCtx, &pb.GetSerialRequest{})
		if err != nil {
			b.notifier.Errorf("Unable to get serial number: %v", err)
			return EventDisconnect
		}

		err = b.rc.EnsureEnrolled(enrollCtx, serial.GetSerial())

		cancel()
		if err != nil {
			b.notifier.Errorf("Bootstrap: %v", err)
			return EventDisconnect
		}
	}

	helperCtx, cancel := context.WithTimeout(ctx, helperTimeout)
	_, err := b.deviceHelper.Configure(helperCtx, b.rc.BuildHelperConfiguration([]*pb.Gateway{
		b.rc.APIServerPeer(),
	}))
	cancel()

	if err != nil {
		b.notifier.Errorf(err.Error())
		return EventDisconnect
	}

	// TODO: status.ConnectedSince = timestamppb.Now()
	return EventBootstrapped
}

func (Bootstrapping) AgentState() pb.AgentState {
	return pb.AgentState_Bootstrapping
}

func (b Bootstrapping) String() string {
	return b.AgentState().String()
}

func (b Bootstrapping) Status() *pb.AgentStatus {
	return &pb.AgentStatus{
		Tenants:         b.baseStatus.GetTenants(),
		ConnectionState: b.AgentState(),
	}
}
