package statemachine

import (
	"context"
)

type Authenticating struct {
}

func (a *Authenticating) Enter(ctx context.Context, sendEvent func(Event)) {
	sendEvent(EventAuthenticated)
}
