package config

import "fmt"

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
}

func (c Config) Valid() error {
	if len(c.Azure.DiscoveryURL) == 0 {
		return fmt.Errorf("--azure-discovery-url must be set")
	}

	if len(c.Azure.ClientID) == 0 {
		return fmt.Errorf("--azure-client-id must be set")
	}

	return nil
}

type Azure struct {
	ClientID     string
	DiscoveryURL string
}

func DefaultConfig() Config {
	return Config{
		BindAddress: "10.255.240.1:80",
		ConfigDir:   "/usr/local/etc/nais-device/",
	}
}
