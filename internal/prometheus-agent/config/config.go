package config

import (
	"fmt"
	"io"
	"net/netip"

	"github.com/nais/device/internal/ioconvenience"
)

type Config struct {
	APIServerPassword   string
	APIServerPublicKey  string
	APIServerTunnelIP   string
	APIServerURL        string
	APIServerUsername   string
	APIServerEndpoint   string
	LogLevel            string
	PrivateKey          string
	PrometheusAddress   string
	DeviceIPv4          string `envconfig:"TUNNELIP"`
	DeviceIPv6          string `envconfig:"TUNNELIPV6"`
	WireGuardEnabled    bool
	WireGuardConfigPath string
	WireGuardIPv4       *netip.Prefix `ignored:"true"`
	WireGuardIPv6       *netip.Prefix `ignored:"true"`
}

func DefaultConfig() Config {
	return Config{
		APIServerURL:        "127.0.0.1:8099",
		PrometheusAddress:   "127.0.0.1:3000",
		LogLevel:            "info",
		WireGuardConfigPath: "/run/wg0.conf",
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

func (cfg Config) WriteWireGuardBase(w io.Writer) error {
	ew := ioconvenience.NewErrorWriter(w)

	_, _ = io.WriteString(ew, "[Interface]\n")
	_, _ = io.WriteString(ew, fmt.Sprintf("PrivateKey = %s\n", cfg.PrivateKey))
	_, _ = io.WriteString(ew, "ListenPort = 51820\n")
	_, _ = io.WriteString(ew, "\n")
	_, _ = io.WriteString(ew, "[Peer] # apiserver\n")
	_, _ = io.WriteString(ew, fmt.Sprintf("PublicKey = %s\n", cfg.APIServerPublicKey))
	_, _ = io.WriteString(ew, fmt.Sprintf("AllowedIPs = %s/32\n", cfg.APIServerTunnelIP))
	_, _ = io.WriteString(ew, fmt.Sprintf("Endpoint = %s\n", cfg.APIServerEndpoint))
	_, _ = io.WriteString(ew, "\n")

	_, err := ew.Status()

	return err
}

func (cfg Config) GetPassword() string {
	return cfg.APIServerPassword
}

func (cfg Config) GetUsername() string {
	return cfg.APIServerUsername
}

func (cfg Config) GetWireGuardConfigPath() string {
	return cfg.WireGuardConfigPath
}
