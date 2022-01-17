package gateway_agent

import (
	"io/ioutil"
	"path"
	"path/filepath"

	"github.com/nais/device/pkg/bootstrap"
)

type Config struct {
	Name                string
	ConfigDir           string
	WireGuardConfigPath string
	BootstrapConfigPath string
	BootstrapApiURL     string
	PrivateKeyPath      string
	PrivateKey          string
	EnableRouting       bool
	PrometheusAddr      string
	PrometheusPublicKey string
	PrometheusTunnelIP  string
	APIServerURL        string
	APIServerPassword   string
	LogLevel            string
	BootstrapConfig     *bootstrap.Config
	PublicIP            string
	EnrollmentToken     string
}

func DefaultConfig() Config {
	cfg := Config{
		APIServerURL:    "127.0.0.1:8099",
		BootstrapApiURL: "https://bootstrap.device.nais.io",
		ConfigDir:       "/etc/gateway-agent/",
		LogLevel:        "info",
		Name:            "test01",
		PrometheusAddr:  "127.0.0.1:3000",
	}

	return cfg
}

func (c *Config) InitLocalConfig() {
	c.WireGuardConfigPath = path.Join("/", "run", "wg0.conf")
	c.PrivateKeyPath = path.Join(c.ConfigDir, "private.key")
	c.BootstrapConfigPath = filepath.Join(c.ConfigDir, "bootstrapconfig.json")

	c.PrivateKey, _ = readFileToString(c.PrivateKeyPath)
}

func readFileToString(filePath string) (string, error) {
	b, err := ioutil.ReadFile(filePath)
	return string(b), err
}
