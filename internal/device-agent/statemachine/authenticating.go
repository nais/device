package statemachine

import (
	"github.com/nais/device/internal/pb"
)

type Authenticating struct {
}

func (a *Authenticating) Enter() {
	//TODO implement me
}

func (a *Authenticating) Exit() {
}

func (a *Authenticating) GetAgentState() pb.AgentState {
	return pb.AgentState_Authenticating
}
