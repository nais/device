package statemachine

import (
	"context"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"

	"github.com/nais/device/internal/pb"
)

type Disconnected struct {
	rc                   runtimeconfig.RuntimeConfig
	cfg                  config.Config
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
	return EventNoOp
}

func (Disconnected) AgentState() pb.AgentState {
	return pb.AgentState_Disconnected
}

func (d Disconnected) String() string {
	return d.AgentState().String()
}
