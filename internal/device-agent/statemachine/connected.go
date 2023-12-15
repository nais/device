package statemachine

import (
	"context"
)

type Connected struct {
}

func (c *Connected) Enter(ctx context.Context, sendEvent func(Event)) {
	<-ctx.Done()
	sendEvent(EventDisconnect)
}
