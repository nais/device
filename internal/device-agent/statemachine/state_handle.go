package statemachine

import (
	"context"
	"sync"
)

type stateHandle struct {
	state      State
	ctx        context.Context
	cancelFunc context.CancelFunc
	mutex      *sync.Mutex
}

func newStateHandle(ctx context.Context, state State) *stateHandle {
	ctx, cancel := context.WithCancel(ctx)
	return &stateHandle{
		state:      state,
		cancelFunc: cancel,
		ctx:        ctx,
		mutex:      &sync.Mutex{},
	}
}

func (sh *stateHandle) enter(out chan<- Event) {
	sh.mutex.Lock()
	go func() {
		out <- sh.state.Enter(sh.ctx)
		sh.mutex.Unlock()
	}()
}

func (sh *stateHandle) exit() {
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

func (sh *stateHandle) String() string {
	return sh.state.String()
}
