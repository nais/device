package config

import (
	"fmt"
	"strings"
)

type Config struct {
	DbConnURI           string
	SlackToken          string
	BindAddress         string
	ConfigDir           string
	PrivateKeyPath      string
	WireGuardConfigPath string
	DevMode             bool
	Endpoint            string
	Azure               Azure
	PrometheusAddr      string
	PrometheusPublicKey string
	PrometheusTunnelIP  string
	APIKeyEntries       []string
}

type Azure struct {
	ClientID     string
	DiscoveryURL string
}

func (c *Config) APIKeys() (map[string]string, error) {
	apiKeys := make(map[string]string)
	for _, key := range c.APIKeyEntries {
		entry := strings.Split(key, ":")
		if len(entry) > 2 {
			return nil, fmt.Errorf("invalid format on apikeys, should be comma-separated entries on format 'user:key'")
		}

		apiKeys[entry[0]] = entry[1]
	}

	if len(apiKeys) == 0 {
		return nil, fmt.Errorf("no API keys provided")
	}

	return apiKeys, nil
}

func DefaultConfig() Config {
	return Config{
		BindAddress:    "10.255.240.1:80",
		ConfigDir:      "/usr/local/etc/nais-device/",
		PrometheusAddr: ":3000",
	}
}
