package config

import (
	"fmt"
	"net/netip"
)

type Config struct {
	APIServerPassword  string
	APIServerPublicKey string
	APIServerTunnelIP  string
	APIServerURL       string
	APIServerUsername  string
	APIServerEndpoint  string
	LogLevel           string
	PrivateKey         string
	PrometheusAddress  string
	DeviceIPv4         string `envconfig:"TUNNELIP"`
	DeviceIPv6         string `envconfig:"TUNNELIPV6"`
	WireGuardEnabled   bool
	WireGuardIPv4      *netip.Prefix `ignored:"true"`
	WireGuardIPv6      *netip.Prefix `ignored:"true"`
}

func DefaultConfig() Config {
	return Config{
		APIServerURL:      "127.0.0.1:8099",
		PrometheusAddress: "127.0.0.1:3000",
		LogLevel:          "info",
	}
}

func (c *Config) Parse() error {
	v4prefix, err := netip.ParsePrefix(c.DeviceIPv4)
	if err != nil {
		return fmt.Errorf("parsing ipv4 prefix: %w", err)
	}
	c.WireGuardIPv4 = &v4prefix

	if len(c.DeviceIPv6) == 0 {
		v6prefix, err := netip.ParsePrefix(c.DeviceIPv6)
		if err != nil {
			return fmt.Errorf("parsing ipv6 prefix: %w", err)
		}
		c.WireGuardIPv6 = &v6prefix
	}

	return nil
}

func (c Config) ValidateWireGuard() error {
	var err error

	check := func(key, value string) error {
		if err != nil {
			return err
		}
		if len(value) == 0 {
			err = fmt.Errorf("missing required configuration option '%s'", key)
		}
		return err
	}

	err = check("apiserver-endpoint", c.APIServerEndpoint)
	err = check("apiserver-password", c.APIServerPassword)
	err = check("apiserver-public-key", c.APIServerPublicKey)
	err = check("apiserver-tunnel-ip", c.APIServerTunnelIP)
	err = check("private-key", c.PrivateKey)

	return err
}

func (cfg Config) GetPassword() string {
	return cfg.APIServerPassword
}

func (cfg Config) GetUsername() string {
	return cfg.APIServerUsername
}
