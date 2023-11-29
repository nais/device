package statemachine

import (
	"github.com/nais/device/internal/pb"
)

type Connected struct {
}

func (c *Connected) Enter() {
	//TODO implement me
}

func (c *Connected) Exit() {
}

func (c *Connected) GetAgentState() pb.AgentState {
	return pb.AgentState_Connected
}
