package state

import (
	"context"

	"github.com/nais/device/internal/pb"
)

type State interface {
	Enter(context.Context) EventWithSpan
	String() string
	Status() *pb.AgentStatus
}
