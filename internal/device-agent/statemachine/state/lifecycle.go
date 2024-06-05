package state

import (
	"context"
	"sync"

	"github.com/nais/device/internal/pb"
)

type StateLifecycle struct {
	name   string
	state  State
	status *pb.AgentStatus

	ctx        context.Context
	cancelFunc context.CancelFunc
	mutex      *sync.Mutex
}

func NewStateLifecycle(ctx context.Context, state State) *StateLifecycle {
	ctx, cancel := context.WithCancel(ctx)
	return &StateLifecycle{
		state:  state,
		name:   state.String(), // cache as it's not thread safe
		status: state.Status(), // cache as it's not thread safe

		cancelFunc: cancel,
		ctx:        ctx,
		mutex:      &sync.Mutex{},
	}
}

func (s *StateLifecycle) Enter(out chan<- EventWithSpan) {
	s.mutex.Lock()
	go func() {
		out <- s.state.Enter(s.ctx)
		s.mutex.Unlock()
	}()
}

func (s *StateLifecycle) Exit() {
	if s.cancelFunc == nil {
		panic("Current state has no cancel function, this is a programmer error")
	}

	s.cancelFunc()
	// Wait for unlock (Enter returns) before we continue in this routine.
	s.mutex.Lock()
}

func (s *StateLifecycle) String() string {
	return s.name
}

func (s *StateLifecycle) Status() *pb.AgentStatus {
	return s.status
}

func (s *StateLifecycle) IsState(state State) bool {
	return state == s.state
}
