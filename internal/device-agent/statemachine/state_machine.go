package statemachine

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/nais/device/internal/pb"
)

type Event string

const (
	EventWaitForExternalEvent Event = "WaitForExternalEvent"
	EventLogin                Event = "Login"
	EventAuthenticated        Event = "Authenticated"
	EventBootstrapped         Event = "Bootstrapped"
	EventDisconnect           Event = "Disconnect"
)

type State interface {
	Enter(context.Context) Event
	AgentState() pb.AgentState
	Status() *pb.AgentStatus
	String() string
}

type StateMachine struct {
	ctx           context.Context
	current       *stateLifecycle
	events        chan Event
	initialState  State
	transitions   map[Event]Transitions
	logger        logrus.FieldLogger
	statusUpdates chan<- *pb.AgentStatus
}

type Transitions struct {
	Target  State
	Sources []State
}

func New(
	ctx context.Context,
	transitions map[Event]Transitions,
	initialState State,
	statusUpdates chan<- *pb.AgentStatus,
	logger logrus.FieldLogger,
) *StateMachine {
	stateMachine := StateMachine{
		ctx:           ctx,
		events:        make(chan Event, 255),
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
	}
}

func (sm *StateMachine) SendEvent(e Event) {
	sm.events <- e
}

func (sm *StateMachine) Run(ctx context.Context) {
	sm.setState(sm.initialState)

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():

		case event := <-sm.events:
			if ctx.Err() != nil {
				break
			}

			sm.logger.Infof("Event received: %s", event)
			if event == EventWaitForExternalEvent {
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

func (sm *StateMachine) setState(state State) {
	sm.logger.Infof("Exiting state: %v", sm.current)
	sm.current.exit()

	sm.current = newStateLifecycle(sm.ctx, state)
	sm.triggerStatusUpdate()

	sm.logger.Infof("Entering state: %v", sm.current)
	sm.current.enter(sm.events)
}

func (sm *StateMachine) transition(event Event) {
	t, ok := sm.transitions[event]
	if !ok {
		sm.logger.Warnf("No defined transitions for event: %s", event)
	}

	for _, s := range t.Sources {
		if s == sm.current.state {
			sm.setState(t.Target)
			return
		}
	}

	sm.logger.Warnf("No defined transition for event %s in state %s", event, sm.GetAgentState())
}
