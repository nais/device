package statemachine

import (
	"context"
	"time"

	"github.com/nais/device/internal/device-agent/auth"
	"github.com/nais/device/internal/pb"
)

const (
	authFlowTimeout = 1 * time.Minute // total timeout for authenticating user (AAD login in browser, redirect to localhost, exchange code for token)
)

type Authenticating struct {
	BaseState
}

func (a *Authenticating) Enter(ctx context.Context) Event {
	session, _ := a.rc.GetTenantSession()
	if !session.Expired() {
		return EventAuthenticated
	}

	ctx, cancel := context.WithTimeout(ctx, authFlowTimeout)
	oauth2Config := a.cfg.OAuth2Config(a.rc.GetActiveTenant().AuthProvider)
	token, err := auth.GetDeviceAgentToken(ctx, a.logger, oauth2Config, a.cfg.GoogleAuthServerAddress)
	cancel()
	if err != nil {
		a.notifier.Errorf("Get token: %v", err)
		return EventDisconnect
	}

	a.rc.SetToken(token)
	return EventAuthenticated
}

func (Authenticating) AgentState() pb.AgentState {
	return pb.AgentState_Authenticating
}

func (a Authenticating) String() string {
	return a.AgentState().String()
}

func (a Authenticating) Status() *pb.AgentStatus {
	return &pb.AgentStatus{
		Tenants:         a.baseStatus.GetTenants(),
		ConnectionState: a.AgentState(),
	}
}
