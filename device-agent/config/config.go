package config

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

type Config struct {
	APIServer           string
	Interface           string
	ConfigDir           string
	BinaryDir           string
	BootstrapToken      string
	WireGuardBinary     string
	WireGuardGoBinary   string
	PrivateKeyPath      string
	WireGuardConfigPath string
	BootstrapTokenPath  string
	BootstrapConfig     *BootstrapConfig
	LogLevel            string
	OAuth2Config        oauth2.Config
	Platform            string
}

type BootstrapConfig struct {
	TunnelIP    string `json:"deviceIP"`
	PublicKey   string `json:"publicKey"`
	Endpoint    string `json:"tunnelEndpoint"`
	APIServerIP string `json:"apiServerIP"`
}

func DefaultConfig() Config {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Getting user conig dir: %w", err)
	}

	return Config{
		APIServer: "http://10.255.240.1",
		Interface: "utun69",
		ConfigDir: filepath.Join(userConfigDir, "nais-device"),
		BinaryDir: "/usr/local/bin",
		LogLevel:  "info",
		OAuth2Config: oauth2.Config{
			ClientID:    "8086d321-c6d3-4398-87da-0d54e3d93967",
			Scopes:      []string{"openid", "6e45010d-2637-4a40-b91d-d4cbb451fb57/.default", "offline_access"},
			Endpoint:    endpoints.AzureAD("62366534-1ec3-4962-8869-9b5535279d0b"),
			RedirectURL: "http://localhost:51800",
		},
	}
}
