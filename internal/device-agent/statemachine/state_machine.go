package statemachine

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	"github.com/nais/device/internal/device-agent/statemachine/state"
	"github.com/nais/device/internal/pb"
)

type StateMachine struct {
	ctx           context.Context
	current       *state.Lifecycle
	events        chan state.EventWithSpan
	initialState  state.State
	transitions   map[state.EventType]Transitions
	logger        logrus.FieldLogger
	statusUpdates chan<- *pb.AgentStatus
}

type Transitions struct {
	Target  state.State
	Sources []state.State
}

func New(
	ctx context.Context,
	transitions map[state.EventType]Transitions,
	initialState state.State,
	statusUpdates chan<- *pb.AgentStatus,
	logger logrus.FieldLogger,
) *StateMachine {
	stateMachine := StateMachine{
		ctx:           ctx,
		events:        make(chan state.EventWithSpan, 255),
		logger:        logger,
		statusUpdates: statusUpdates,
		transitions:   transitions,
		initialState:  initialState,
	}

	for e, t := range stateMachine.transitions {
		if t.Target == nil {
			panic(fmt.Sprintf("transition with nil state detected for event: %v", e))
		}
		for _, s := range t.Sources {
			if s == nil {
				panic(fmt.Sprintf("transition with nil source detected for event: %v", e))
			}
		}
	}

	return &stateMachine
}

func (sm *StateMachine) triggerStatusUpdate() {
	select {
	case sm.statusUpdates <- sm.current.Status():
	default:
		sm.logger.Warn("failed to trigger status update, channel full")
	}
}

func (sm *StateMachine) SendEvent(e state.EventWithSpan) {
	sm.events <- e
}

func (sm *StateMachine) Run(ctx context.Context) {
	sm.setState(ctx, sm.initialState)

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():

		case event := <-sm.events:
			if ctx.Err() != nil {
				break
			}

			sm.logger.WithField("event", event).Info("event received")
			if event.Event == state.EventWaitForExternalEvent {
				continue
			}

			sm.transition(event)
		}
	}

	sm.current.Exit()
}

func (sm *StateMachine) setState(ctx context.Context, s state.State) {
	if sm.current != nil {
		sm.logger.WithField("current_state", sm.current).Info("exiting state")
		sm.current.Exit()
	}

	sm.current = state.NewLifecycle(ctx, s)
	sm.triggerStatusUpdate()

	sm.logger.WithField("new_state", sm.current).Info("entering state")
	sm.current.Enter(sm.events)
}

func (sm *StateMachine) transition(event state.EventWithSpan) {
	t, ok := sm.transitions[event.Event]
	if !ok {
		sm.logger.WithField("event", event).Warn("no defined transitions for event")
	}

	ctx := trace.ContextWithSpan(sm.ctx, event.Span)
	for _, s := range t.Sources {
		if sm.current.IsState(s) {
			sm.setState(ctx, t.Target)
			return
		}
	}

	sm.logger.WithField("event", event).WithField("state", sm.current).Warn("no defined transition for event in state")
	sm.triggerStatusUpdate()
}
