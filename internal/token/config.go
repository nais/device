package token

import (
	"fmt"
	"strings"
)

type Config struct {
	ClientID       string
	Issuer         string
	Endpoint       string
	AllowedDomains []string
}

func (c Config) Validate() error {
	if len(c.ClientID) == 0 {
		return fmt.Errorf("client ID is required")
	}
	if len(c.Issuer) == 0 {
		return fmt.Errorf("issuer is required")
	}
	if len(c.Endpoint) == 0 {
		return fmt.Errorf("endpoint is required")
	}

	if strings.Contains(c.Issuer, "googleapis.com") && len(c.AllowedDomains) == 0 {
		return fmt.Errorf("at least one allowed domain is required for Google tokens")
	}

	return nil
}
