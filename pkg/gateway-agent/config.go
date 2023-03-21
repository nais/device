package gateway_agent

import "C"
import (
	"fmt"

	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/wireguard"
)

type Config struct {
	APIServerEndpoint   string
	APIServerPassword   string
	APIServerPrivateIP  string
	APIServerPublicKey  string
	APIServerURL        string
	ConfigDir           string
	DeviceIP            string
	EnableRouting       bool
	LogLevel            string
	Name                string
	PrivateKey          string
	PrometheusAddr      string
	PrometheusPublicKey string
	PrometheusTunnelIP  string
	WireGuardConfigPath string
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
	err = check("device-ip", c.DeviceIP)
	err = check("private-key", c.PrivateKey)

	return err
}

func (c Config) StaticPeers() []wireguard.Peer {
	peers := []wireguard.Peer{
		&pb.Gateway{
			Name:      "apiserver",
			PublicKey: c.APIServerPublicKey,
			Endpoint:  c.APIServerEndpoint,
			Ip:        c.APIServerPrivateIP,
		},
	}

	if c.PrometheusTunnelIP != "" {
		peers = append(peers,
			&pb.Gateway{
				Name:      "prometheus",
				PublicKey: c.PrometheusPublicKey,
				Ip:        c.PrometheusTunnelIP,
			})
	}

	return peers
}
