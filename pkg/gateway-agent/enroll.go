package gateway_agent

import "github.com/nais/device/pkg/gateway-agent/config"

const (
	DefaultConfigPath = "/etc/default/gateway-agent"
)

type Enroller struct {
	cfg config.Config
}

func NewEnroller(cfg config.Config) Enroller {
	return Enroller{
		cfg: cfg,
	}
}
