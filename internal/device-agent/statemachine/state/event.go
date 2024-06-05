package state

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type EventType string

type EventWithSpan struct {
	Event EventType
	Span  trace.Span
}

func (e EventWithSpan) String() string {
	return string(e.Event)
}

const (
	EventWaitForExternalEvent EventType = "WaitForExternalEvent"
	EventLogin                EventType = "Login"
	EventAuthenticated        EventType = "Authenticated"
	EventBootstrapped         EventType = "Bootstrapped"
	EventDisconnect           EventType = "Disconnect"
)

func SpanEvent(ctx context.Context, e EventType) EventWithSpan {
	return EventWithSpan{
		Event: e,
		Span:  trace.SpanFromContext(ctx),
	}
}
