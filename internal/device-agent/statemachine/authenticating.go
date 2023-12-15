package statemachine

import (
	"context"
	"github.com/nais/device/internal/pb"
)

type Authenticating struct {
}

func (a *Authenticating) Enter(ctx context.Context) {
	// configure oauth parameters
	// start http server
	// wait for either http server response OR ctx.Done
	// stop http server
	// if done:
	//	return
	// if response:
	//	exchange code for token
	//	wait for response or ctx.Done (handled by http client, just make sure to pass ctx)
	// return
}

func (a *Authenticating) Exit(context.Context) {
}

func (a *Authenticating) GetAgentState() pb.AgentState {
	return pb.AgentState_Authenticating
}
