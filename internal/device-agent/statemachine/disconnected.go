package statemachine

import (
	"context"
	"github.com/nais/device/internal/pb"
)

type Disconnected struct {
}

func (d *Disconnected) Enter(context.Context) {
	// TODO: implement disconnection logic
}

func (d *Disconnected) Exit(context.Context) {
}

func (d *Disconnected) GetAgentState() pb.AgentState {
	return pb.AgentState_Disconnected
}
