package config

import (
	"fmt"
	"strings"
)

type Config struct {
	DbConnURI               string
	BootstrapApiCredentials string
	BindAddress             string
	ConfigDir               string
	PrivateKeyPath          string
	WireGuardConfigPath     string
	DevMode                 bool
	Endpoint                string
	Azure                   Azure
	PrometheusAddr          string
	PrometheusPublicKey     string
	PrometheusTunnelIP      string
	CredentialEntries       []string
	BootstrapApiURL         string
	LogLevel                string
}

type Azure struct {
	ClientID     string
	DiscoveryURL string
}

func (c *Config) Credentials() (map[string]string, error) {
	credentials := make(map[string]string)
	for _, key := range c.CredentialEntries {
		entry := strings.Split(key, ":")
		if len(entry) > 2 {
			return nil, fmt.Errorf("invalid format on credentials, should be comma-separated entries on format 'user:key'")
		}

		credentials[entry[0]] = entry[1]
	}

	return credentials, nil
}

func DefaultConfig() Config {
	return Config{
		BindAddress:    "10.255.240.1:80",
		ConfigDir:      "/usr/local/etc/nais-device/",
		PrometheusAddr: ":3000",
		LogLevel: "info",
	}
}
