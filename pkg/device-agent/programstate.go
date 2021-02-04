package device_agent

import (
	"fmt"
	"time"
)

type ProgramState int

const (
	StateDisconnected ProgramState = iota
	StateNewVersion
	StateBootstrapping
	StateConnected
	StateDisconnecting
	StateUnhealthy
	StateQuitting
	StateAuthenticating
	StateSyncConfig
	StateHealthCheck
)

const (
	versionCheckInterval      = 1 * time.Hour
	syncConfigInterval        = 5 * time.Minute
	initialGatewayRefreshWait = 2 * time.Second
	initialConnectWait        = initialGatewayRefreshWait
	healthCheckInterval       = 20 * time.Second
)

func (state ProgramState) String() string {
	switch state {
	case StateDisconnected:
		return "Disconnected"
	case StateBootstrapping:
		return "Bootstrapping..."
	case StateAuthenticating:
		return "Authenticating..."
	case StateSyncConfig:
		return "Synchronizing configuration..."
	case StateHealthCheck:
		return "Checking gateway health..."
	case StateConnected:
		return fmt.Sprintf("Connected since %s", connectedTime.Format(time.Kitchen))
	case StateUnhealthy:
		return "Device is unhealthy >_<"
	case StateDisconnecting:
		return "Disconnecting..."
	case StateQuitting:
		return "Quitting..."
	default:
		return "Unknown state >_<"
	}
}
