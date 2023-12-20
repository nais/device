package statemachine

import (
	"context"
	"sync"
)

type stateLifeCycle struct {
	state      State
	ctx        context.Context
	cancelFunc context.CancelFunc
	mutex      *sync.Mutex
}

func newStateLifecycle(ctx context.Context, state State) *stateLifeCycle {
	ctx, cancel := context.WithCancel(ctx)
	return &stateLifeCycle{
		state:      state,
		cancelFunc: cancel,
		ctx:        ctx,
		mutex:      &sync.Mutex{},
	}
}

func (sh *stateLifeCycle) enter(out chan<- Event) {
	sh.mutex.Lock()
	go func() {
		out <- sh.state.Enter(sh.ctx)
		sh.mutex.Unlock()
	}()
}

func (sh *stateLifeCycle) exit() {
	if sh == nil {
		return
	}

	if sh.cancelFunc == nil {
		panic("Current state has no cancel function, this is a programmer error")
	}

	sh.cancelFunc()
	// Wait for unlock (Enter returns) before we continue in this routine.
	sh.mutex.Lock()
}

func (sh *stateLifeCycle) String() string {
	return sh.state.String()
}
