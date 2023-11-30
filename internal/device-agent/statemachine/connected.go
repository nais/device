package statemachine

import (
	"context"
	"github.com/nais/device/internal/pb"
)

type Connected struct {
}

func (c *Connected) Enter(context.Context) {
	//TODO implement me
}

func (c *Connected) Exit(context.Context) {
}

func (c *Connected) GetAgentState() pb.AgentState {
	return pb.AgentState_Connected
}
