package disconnected

import (
	"context"

	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine"
	"github.com/nais/device/internal/pb"
)

type Disconnected struct {
	rc  runtimeconfig.RuntimeConfig
	cfg config.Config

	autoConnectTriggered bool
}

func New(rc runtimeconfig.RuntimeConfig, cfg config.Config) statemachine.State {
	return &Disconnected{
		rc:  rc,
		cfg: cfg,
	}
}

func (d *Disconnected) Enter(ctx context.Context) statemachine.Event {
	d.rc.SetToken(nil)
	d.rc.ResetEnrollConfig()

	if d.cfg.AgentConfiguration.AutoConnect && !d.autoConnectTriggered {
		d.autoConnectTriggered = true
		return statemachine.EventLogin
	}
	<-ctx.Done()
	return statemachine.EventWaitForExternalEvent
}

func (Disconnected) AgentState() pb.AgentState {
	return pb.AgentState_Disconnected
}

func (d Disconnected) String() string {
	return "Disconnected"
}

func (d Disconnected) Status() *pb.AgentStatus {
	return &pb.AgentStatus{
		ConnectionState: d.AgentState(),
	}
}
