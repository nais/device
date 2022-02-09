package prometheusagent

import (
	"fmt"
	"io"

	"github.com/nais/device/pkg/ioconvenience"
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
	PrometheusAddr      string
	TunnelIP            string
	WireGuardConfigPath string
	WireGuardEnabled    bool
}

func DefaultConfig() Config {
	return Config{
		APIServerURL:        "http://127.0.0.1:8099",
		PrometheusAddr:      "127.0.0.1:3000",
		LogLevel:            "info",
		WireGuardConfigPath: "/run/wg0.conf",
	}
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
	err = check("tunnel-ip", c.TunnelIP)
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

func (cfg Config) GetTunnelIP() string {
	return cfg.TunnelIP
}
