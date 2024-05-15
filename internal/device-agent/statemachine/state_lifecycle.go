package statemachine

import (
	"context"
	"sync"
)

type stateLifecycle struct {
	state      State
	ctx        context.Context
	cancelFunc context.CancelFunc
	mutex      *sync.Mutex
}

func newStateLifecycle(ctx context.Context, state State) *stateLifecycle {
	ctx, cancel := context.WithCancel(ctx)
	return &stateLifecycle{
		state:      state,
		cancelFunc: cancel,
		ctx:        ctx,
		mutex:      &sync.Mutex{},
	}
}

func (s *stateLifecycle) enter(out chan<- EventWithSpan) {
	s.mutex.Lock()
	go func() {
		out <- s.state.Enter(s.ctx)
		s.mutex.Unlock()
	}()
}

func (s *stateLifecycle) exit() {
	if s == nil {
		return
	}

	if s.cancelFunc == nil {
		panic("Current state has no cancel function, this is a programmer error")
	}

	s.cancelFunc()
	// Wait for unlock (Enter returns) before we continue in this routine.
	s.mutex.Lock()
}

func (s *stateLifecycle) String() string {
	return s.state.String()
}
