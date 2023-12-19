package statemachine

import (
	"context"

	"github.com/nais/device/internal/pb"
)

type Disconnected struct {
}

func (d *Disconnected) Enter(ctx context.Context, sendEvent func(Event)) {
}

func (Disconnected) AgentState() pb.AgentState {
	return pb.AgentState_Disconnected
}

func (d Disconnected) String() string {
	return d.AgentState().String()
}
