package config

import (
	"fmt"
	"net/netip"

	"github.com/nais/device/internal/wireguard"
	"github.com/nais/device/pkg/pb"
)

type Config struct {
	APIServerEndpoint   string
	APIServerPassword   string
	APIServerPrivateIP  string
	APIServerPublicKey  string
	APIServerURL        string
	ConfigDir           string
	DeviceIPv4          string `envconfig:"DEVICEIP"` // Not changing to v4 yet as it's configured in config files on disk all around
	DeviceIPv6          string
	EnableRouting       bool
	LogLevel            string
	Name                string
	PrivateKey          string
	PrometheusAddr      string
	PrometheusPublicKey string
	PrometheusTunnelIP  string
	WireGuardConfigPath string
	WireGuardIPv4       *netip.Prefix `ignored:"true"`
	WireGuardIPv6       *netip.Prefix `ignored:"true"`
	AutoEnroll          bool
}

func DefaultConfig() Config {
	return Config{
		APIServerURL:        "127.0.0.1:8099",
		ConfigDir:           "/etc/gateway-agent/",
		LogLevel:            "info",
		Name:                "test01",
		PrometheusAddr:      "127.0.0.1:3000",
		WireGuardConfigPath: "/run/wg0.conf",
	}
}

func (c *Config) Parse() error {
	v4prefix, err := netip.ParsePrefix(c.DeviceIPv4)
	if err != nil {
		return fmt.Errorf("parsing ipv4 prefix: %w", err)
	}
	c.WireGuardIPv4 = &v4prefix

	if len(c.DeviceIPv6) > 0 {
		v6prefix, err := netip.ParsePrefix(c.DeviceIPv6)
		if err != nil {
			return fmt.Errorf("parsing ipv6 prefix: %w", err)
		}
		c.WireGuardIPv6 = &v6prefix
	}

	return nil
}

func (c Config) ValidateWireGuard() error {
	// values are provided runtime -- no early validation needed
	if c.AutoEnroll {
		return nil
	}

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
	err = check("apiserver-private-ip", c.APIServerPrivateIP)
	err = check("device-ip", c.DeviceIPv4)
	err = check("private-key", c.PrivateKey)

	return err
}

func (c Config) StaticPeers() []wireguard.Peer {
	return []wireguard.Peer{
		&pb.Gateway{
			Name:      wireguard.APIServerPeerName,
			PublicKey: c.APIServerPublicKey,
			Endpoint:  c.APIServerEndpoint,
			Ipv4:      c.APIServerPrivateIP,
		},
		&pb.Gateway{
			Name:      wireguard.PrometheusPeerName,
			PublicKey: c.PrometheusPublicKey,
			Ipv4:      c.PrometheusTunnelIP,
		},
	}
}
