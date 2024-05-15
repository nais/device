package statemachine

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"

	"github.com/nais/device/internal/pb"
)

type EventWithSpan struct {
	Event EventType
	Span  trace.Span
}

type EventType string

const (
	EventWaitForExternalEvent EventType = "WaitForExternalEvent"
	EventLogin                EventType = "Login"
	EventAuthenticated        EventType = "Authenticated"
	EventBootstrapped         EventType = "Bootstrapped"
	EventDisconnect           EventType = "Disconnect"
)

type State interface {
	Enter(context.Context) EventWithSpan
	AgentState() pb.AgentState
	Status() *pb.AgentStatus
	String() string
}

type StateMachine struct {
	ctx           context.Context
	current       *stateLifecycle
	events        chan EventWithSpan
	initialState  State
	transitions   map[EventType]Transitions
	logger        logrus.FieldLogger
	statusUpdates chan<- *pb.AgentStatus
}

type Transitions struct {
	Target  State
	Sources []State
}

func New(
	ctx context.Context,
	transitions map[EventType]Transitions,
	initialState State,
	statusUpdates chan<- *pb.AgentStatus,
	logger logrus.FieldLogger,
) *StateMachine {
	stateMachine := StateMachine{
		ctx:           ctx,
		events:        make(chan EventWithSpan, 255),
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
	case sm.statusUpdates <- sm.current.state.Status():
	default:
		sm.logger.Warn("failed to trigger status update, channel full")
	}
}

func (sm *StateMachine) SendEvent(e EventWithSpan) {
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

			sm.logger.Infof("Event received: %s", event)
			if event.Event == EventWaitForExternalEvent {
				continue
			}

			sm.transition(event)
		}
	}

	sm.current.exit()
}

func (sm *StateMachine) GetAgentState() pb.AgentState {
	if sm.current == nil {
		return sm.initialState.AgentState()
	}

	return sm.current.state.AgentState()
}

func (sm *StateMachine) setState(ctx context.Context, state State) {
	sm.logger.Infof("Exiting state: %v", sm.current)
	sm.current.exit()

	sm.current = newStateLifecycle(ctx, state)
	sm.triggerStatusUpdate()

	sm.logger.Infof("Entering state: %v", sm.current)
	sm.current.enter(sm.events)
}

func (sm *StateMachine) transition(event EventWithSpan) {
	t, ok := sm.transitions[event.Event]
	if !ok {
		sm.logger.Warnf("No defined transitions for event: %s", event)
	}

	ctx := trace.ContextWithSpan(sm.ctx, event.Span)
	for _, s := range t.Sources {
		if s == sm.current.state {
			sm.setState(ctx, t.Target)
			return
		}
	}

	sm.logger.Warnf("No defined transition for event %s in state %s", event, sm.GetAgentState())
	sm.triggerStatusUpdate()
}

func SpanEvent(ctx context.Context, e EventType) EventWithSpan {
	return EventWithSpan{
		Event: e,
		Span:  trace.SpanFromContext(ctx),
	}
}
