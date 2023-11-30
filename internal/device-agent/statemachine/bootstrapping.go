package statemachine

import (
	"context"
	"github.com/nais/device/internal/pb"
)

type Bootstrapping struct {
}

func (b *Bootstrapping) Enter(context.Context) {
	//TODO implement me
}

func (b *Bootstrapping) Exit(context.Context) {
}

func (b *Bootstrapping) GetAgentState() pb.AgentState {
	return pb.AgentState_Bootstrapping
}
