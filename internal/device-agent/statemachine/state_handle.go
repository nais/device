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
		maybeE := sh.state.Enter(sh.ctx)
		if maybeE != EventWaitForExternalEvent {
			out <- maybeE
		}
		sh.mutex.Unlock()
	}()
}

func (sh *stateHandle) exit() {
	if sh == nil {
		return
	}

	if sh.cancelFunc != nil {
		sh.cancelFunc()
		// Wait for unlock (Enter returns) before we continue in this routine.
		sh.mutex.Lock()
	} else {
		panic("Current state has no cancel function, this is a programmer error")
	}
}

func (sh *stateHandle) String() string {
	return sh.state.String()
}
