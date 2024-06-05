package state

import (
	"context"
	"sync"

	"github.com/nais/device/internal/pb"
)

type Lifecycle struct {
	name   string
	state  State
	status *pb.AgentStatus

	ctx        context.Context
	cancelFunc context.CancelFunc
	mutex      *sync.Mutex
}

func NewLifecycle(ctx context.Context, state State) *Lifecycle {
	ctx, cancel := context.WithCancel(ctx)
	return &Lifecycle{
		state:  state,
		name:   state.String(), // cache as it's not thread safe
		status: state.Status(), // cache as it's not thread safe

		cancelFunc: cancel,
		ctx:        ctx,
		mutex:      &sync.Mutex{},
	}
}

func (s *Lifecycle) Enter(out chan<- EventWithSpan) {
	s.mutex.Lock()
	go func() {
		out <- s.state.Enter(s.ctx)
		s.mutex.Unlock()
	}()
}

func (s *Lifecycle) Exit() {
	if s.cancelFunc == nil {
		panic("Current state has no cancel function, this is a programmer error")
	}

	s.cancelFunc()
	// Wait for unlock (Enter returns) before we continue in this routine.
	s.mutex.Lock()
}

func (s *Lifecycle) String() string {
	return s.name
}

func (s *Lifecycle) Status() *pb.AgentStatus {
	return s.status
}

func (s *Lifecycle) IsState(state State) bool {
	return state == s.state
}
