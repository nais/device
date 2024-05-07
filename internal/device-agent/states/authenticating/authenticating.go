package authenticating

import (
	"context"
	"time"

	"github.com/nais/device/internal/device-agent/auth"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/otel"
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
	getToken auth.GetTokenFunc
	logger   logrus.FieldLogger
}

func New(rc runtimeconfig.RuntimeConfig, cfg config.Config, logger logrus.FieldLogger, notifier notify.Notifier) statemachine.State {
	return &Authenticating{
		rc:       rc,
		cfg:      cfg,
		logger:   logger,
		notifier: notifier,

		getToken: auth.GetDeviceAgentToken,
	}
}

func (a *Authenticating) Enter(ctx context.Context) statemachine.Event {
	ctx, span := otel.Start(ctx, "Authenticating")
	defer span.End()

	session, _ := a.rc.GetTenantSession()
	if !session.Expired() {
		span.AddEvent("session.active")
		return statemachine.EventAuthenticated
	}

	ctx, cancel := context.WithTimeout(ctx, authFlowTimeout)
	oauth2Config := a.cfg.OAuth2Config(a.rc.GetActiveTenant().AuthProvider)
	token, err := a.getToken(ctx, a.logger, oauth2Config, a.cfg.GoogleAuthServerAddress)
	cancel()
	if err != nil {
		span.RecordError(err)
		a.notifier.Errorf("Get token: %v", err)
		return statemachine.EventDisconnect
	}

	span.AddEvent("session.new")
	a.rc.SetToken(token)
	return statemachine.EventAuthenticated
}

func (Authenticating) AgentState() pb.AgentState {
	return pb.AgentState_Authenticating
}

func (a Authenticating) String() string {
	return "Authenticating"
}

func (a Authenticating) Status() *pb.AgentStatus {
	return &pb.AgentStatus{
		ConnectionState: a.AgentState(),
	}
}
