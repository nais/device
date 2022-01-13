package gateway_agent

import (
	"io/ioutil"

	"github.com/nais/device/pkg/bootstrap"
)

type Config struct {
	Name                  string
	ConfigDir             string
	WireGuardConfigPath   string
	BootstrapConfigPath   string
	BootstrapApiURL       string
	PrivateKeyPath        string
	PrivateKey            string
	EnableRouting         bool
	PrometheusAddr        string
	PrometheusPublicKey   string
	PrometheusTunnelIP    string
	APIServerURL          string
	APIServerPassword     string
	APIServerPasswordPath string
	LogLevel              string
	BootstrapConfig       *bootstrap.Config
	PublicIP              string
	EnrollmentToken       string
}

func DefaultConfig() Config {
	return Config{
		APIServerURL:    "127.0.0.1:8099",
		BootstrapApiURL: "https://bootstrap.device.nais.io",
		ConfigDir:       "/usr/local/etc/nais-device",
		LogLevel:        "debug",
		PrometheusAddr:  "127.0.0.1:3000",
	}
}

func (c *Config) InitLocalConfig() error {
	c.PrivateKey, _ = readFileToString(c.PrivateKeyPath)
	c.APIServerPassword, _ = readFileToString(c.APIServerPasswordPath)
	return nil
}

func readFileToString(filePath string) (string, error) {
	b, err := ioutil.ReadFile(filePath)
	return string(b), err
}
