package statemachine

import (
	"context"
	"github.com/nais/device/internal/pb"
)

type Authenticating struct {
}

func (a *Authenticating) Enter(context.Context) {
	//TODO implement me
}

func (a *Authenticating) Exit(context.Context) {
}

func (a *Authenticating) GetAgentState() pb.AgentState {
	return pb.AgentState_Authenticating
}
