package gateway_agent

import (
	"fmt"
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
		ConfigDir:       "/usr/local/etc/nais-device",
		PrometheusAddr:  ":3000",
		BootstrapApiURL: "https://bootstrap.device.nais.io",
		LogLevel:        "info",
	}
}

func (c *Config) InitLocalConfig() error {
	var err error
	c.PrivateKey, err = readFileToString(c.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("reading private key: %s", err)
	}
	c.APIServerPassword, err = readFileToString(c.APIServerPasswordPath)
	if err != nil {
		return fmt.Errorf("reading API server password: %s", err)
	}
	if len(c.APIServerPassword) == 0 {
		return fmt.Errorf("API server password file empty: %s", c.APIServerPasswordPath)
	}

	return nil
}

func readFileToString(filePath string) (string, error) {
	b, err := ioutil.ReadFile(filePath)
	return string(b), err
}
