package statemachine

import (
	"context"

	"github.com/nais/device/internal/pb"
)

type Disconnected struct {
	BaseState
	autoConnectTriggered bool
}

func (d *Disconnected) Enter(ctx context.Context) Event {
	d.rc.SetToken(nil)
	d.rc.ResetEnrollConfig()

	if d.cfg.AgentConfiguration.AutoConnect && !d.autoConnectTriggered {
		d.autoConnectTriggered = true
		return EventLogin
	}
	<-ctx.Done()
	return EventWaitForExternalEvent
}

func (Disconnected) AgentState() pb.AgentState {
	return pb.AgentState_Disconnected
}

func (d Disconnected) String() string {
	return d.AgentState().String()
}

func (d Disconnected) Status() *pb.AgentStatus {
	return &pb.AgentStatus{
		Tenants:         d.baseStatus.GetTenants(),
		ConnectionState: d.AgentState(),
	}
}
