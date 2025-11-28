package disconnected

import (
	"context"

	"github.com/nais/device/internal/deviceagent/config"
	"github.com/nais/device/internal/deviceagent/runtimeconfig"
	"github.com/nais/device/internal/deviceagent/statemachine/state"
	"github.com/nais/device/pkg/pb"
)

type Disconnected struct {
	rc  runtimeconfig.RuntimeConfig
	cfg config.Config

	autoConnectTriggered bool
}

func New(rc runtimeconfig.RuntimeConfig, cfg config.Config) state.State {
	return &Disconnected{
		rc:  rc,
		cfg: cfg,
	}
}

func (d *Disconnected) Enter(ctx context.Context) state.EventWithSpan {
	d.rc.SetToken(nil)
	d.rc.ResetEnrollConfig()
	d.rc.SetAPIServerInfo(nil, "")
	d.rc.SetJitaToken(nil)

	if d.cfg.AgentConfiguration.AutoConnect && !d.autoConnectTriggered {
		d.autoConnectTriggered = true
		return state.SpanEvent(ctx, state.EventLogin)
	}
	<-ctx.Done()
	return state.SpanEvent(ctx, state.EventWaitForExternalEvent)
}

func (d Disconnected) String() string {
	return "Disconnected"
}

func (d Disconnected) Status() *pb.AgentStatus {
	return &pb.AgentStatus{
		ConnectionState: pb.AgentState_Disconnected,
	}
}
