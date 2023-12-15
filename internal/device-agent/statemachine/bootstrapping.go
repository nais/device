package statemachine

import (
	"context"
)

type Bootstrapping struct {
}

func (b *Bootstrapping) Enter(ctx context.Context, sendEvent func(Event)) {
	sendEvent(EventBootstrapped)
}
