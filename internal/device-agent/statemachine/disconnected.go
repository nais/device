package statemachine

import "github.com/nais/device/internal/pb"

type Disconnected struct {
}

func (d *Disconnected) Enter() {
	// TODO: implement disconnection logic
}

func (d *Disconnected) Exit() {
}

func (d *Disconnected) GetAgentState() pb.AgentState {
	return pb.AgentState_Disconnected
}
