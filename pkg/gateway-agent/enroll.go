package gateway_agent

import (
	"encoding/base64"
	"fmt"
	"os"
)

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

func (e *Enroller) persistPassword(password []byte, file string) error {
	fd, err := os.OpenFile(file, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(password)
	_, err = fmt.Fprintf(fd, "GATEWAY_AGENT_APISERVERPASSWORD=\"%s\"\n", encoded)
	if err != nil {
		fd.Close()
		return fmt.Errorf("append to file: %w", err)
	}

	return fd.Close()
}
