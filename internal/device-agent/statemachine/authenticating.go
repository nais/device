package statemachine

import (
	"context"
	"time"

	"github.com/nais/device/internal/device-agent/auth"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
)

const (
	authFlowTimeout = 1 * time.Minute // total timeout for authenticating user (AAD login in browser, redirect to localhost, exchange code for token)
)

type Authenticating struct {
	rc       runtimeconfig.RuntimeConfig
	cfg      config.Config
	notifier notify.Notifier
	logger   logrus.FieldLogger
}

func (a *Authenticating) Enter(ctx context.Context, sendEvent func(Event)) {
	session, _ := a.rc.GetTenantSession()
	if !session.Expired() {
		sendEvent(EventAuthenticated)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, authFlowTimeout)
	oauth2Config := a.cfg.OAuth2Config(a.rc.GetActiveTenant().AuthProvider)
	token, err := auth.GetDeviceAgentToken(ctx, a.logger, oauth2Config, a.cfg.GoogleAuthServerAddress)
	cancel()
	if err != nil {
		a.notifier.Errorf("Get token: %v", err)
		sendEvent(EventDisconnect)
		return
	}

	a.rc.SetToken(token)
	sendEvent(EventAuthenticated)
}

func (Authenticating) AgentState() pb.AgentState {
	return pb.AgentState_Authenticating
}

func (a Authenticating) String() string {
	return a.AgentState().String()
}
