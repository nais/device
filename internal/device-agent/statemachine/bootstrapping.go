package statemachine

import (
	"github.com/nais/device/internal/pb"
)

type Bootstrapping struct {
}

func (b *Bootstrapping) Enter() {
	//TODO implement me
}

func (b *Bootstrapping) Exit() {
}

func (b *Bootstrapping) GetAgentState() pb.AgentState {
	return pb.AgentState_Bootstrapping
}
