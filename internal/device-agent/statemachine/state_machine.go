package statemachine

import (
	"context"
	"fmt"
	"github.com/nais/device/internal/pb"
)

type Event int8

const (
	EventLogin Event = iota
	EventAuthenticated
	EventBootstrapped
	EventDisconnect
)

type State interface {
	Enter(ctx context.Context)
	Exit(ctx context.Context)
	GetAgentState() pb.AgentState
}

type StateMachine struct {
	ctx          context.Context
	cancelFunc   context.CancelFunc
	currentState State
	states       map[pb.AgentState]State
	transitions  map[transitionKey]pb.AgentState
}

type Transitions struct {
	Event       Event
	Sources     []pb.AgentState
	Destination pb.AgentState
}

type transitionKey struct {
	event  Event
	source pb.AgentState
}

func NewStateMachine(ctx context.Context, initialState pb.AgentState, transitions []Transitions, states []State) (*StateMachine, error) {
	ctx, cancelFunc := context.WithCancel(ctx)
	stateMachine := StateMachine{
		ctx:         ctx,
		cancelFunc:  cancelFunc,
		states:      make(map[pb.AgentState]State),
		transitions: make(map[transitionKey]pb.AgentState),
	}
	for _, state := range states {
		stateMachine.states[state.GetAgentState()] = state
	}
	for _, transition := range transitions {
		if stateMachine.states[transition.Destination] == nil {
			return nil, fmt.Errorf("destination state %s not found", transition.Destination)
		}
		for _, source := range transition.Sources {
			if stateMachine.states[source] == nil {
				return nil, fmt.Errorf("source state %s not found", source)
			}
			stateMachine.transitions[transitionKey{transition.Event, source}] = transition.Destination
		}
	}
	stateMachine.currentState = stateMachine.states[initialState]
	stateMachine.currentState.Enter(ctx)
	return &stateMachine, nil
}

func (sm *StateMachine) setState(agentState pb.AgentState) {
	var state State
	state, ok := sm.states[agentState]
	if !ok {
		panic("state not found")
	}
	sm.currentState.Exit(sm.ctx)
	sm.currentState = state
	sm.currentState.Enter(sm.ctx)
}

func (sm *StateMachine) Transition(event Event) {
	key := transitionKey{event, sm.currentState.GetAgentState()}
	if agentState, ok := sm.transitions[key]; ok {
		sm.setState(agentState)
	}
}

func (sm *StateMachine) Close() {
	sm.currentState.Exit(sm.ctx)
	sm.cancelFunc()
}
