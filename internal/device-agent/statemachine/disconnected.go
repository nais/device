package statemachine

import (
	"context"
)

type Disconnected struct {
}

func (d *Disconnected) Enter(ctx context.Context, sendEvent func(Event)) {
}
