package gateway_agent

import (
	"fmt"
	"io"

	"github.com/nais/device/pkg/ioconvenience"
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

func (cfg *Config) WriteWireGuardBase(w io.Writer) error {
	ew := ioconvenience.NewErrorWriter(w)

	_, _ = io.WriteString(ew, "[Interface]\n")
	_, _ = io.WriteString(ew, fmt.Sprintf("PrivateKey = %s\n", cfg.PrivateKey))
	_, _ = io.WriteString(ew, "ListenPort = 51820\n")
	_, _ = io.WriteString(ew, "\n")
	_, _ = io.WriteString(ew, "[Peer] # apiserver\n")
	_, _ = io.WriteString(ew, fmt.Sprintf("PublicKey = %s\n", cfg.APIServerPublicKey))
	_, _ = io.WriteString(ew, fmt.Sprintf("AllowedIPs = %s/32\n", cfg.APIServerPrivateIP))
	_, _ = io.WriteString(ew, fmt.Sprintf("Endpoint = %s\n", cfg.APIServerEndpoint))
	_, _ = io.WriteString(ew, "\n")
	_, _ = io.WriteString(ew, "[Peer] # prometheus\n")
	_, _ = io.WriteString(ew, fmt.Sprintf("PublicKey = %s\n", cfg.PrometheusPublicKey))
	_, _ = io.WriteString(ew, fmt.Sprintf("AllowedIPs = %s/32\n", cfg.PrometheusTunnelIP))
	_, _ = io.WriteString(ew, "\n")

	_, err := ew.Status()

	return err
}
