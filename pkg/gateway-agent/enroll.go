package gateway_agent

const (
	DefaultConfigPath = "/etc/default/gateway-agent"
)

type Enroller struct {
	cfg Config
}

func NewEnroller(cfg Config) Enroller {
	return Enroller{
		cfg: cfg,
	}
}
