package config

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt"
)

const NaisDeviceApprovalGroup = "ffd89425-c75c-4618-b5ab-67149ddbbc2d"

type Config struct {
	DbConnDSN                     string
	BootstrapApiCredentials       string
	BindAddress                   string
	ConfigDir                     string
	PrivateKeyPath                string
	WireGuardConfigPath           string
	DevMode                       bool
	Endpoint                      string
	Azure                         Azure
	PrometheusAddr                string
	PrometheusPublicKey           string
	PrometheusTunnelIP            string
	CredentialEntries             []string
	BootstrapAPIURL               string
	LogLevel                      string
	TokenValidator                jwt.Keyfunc
	GatewayConfigBucketName       string
	GatewayConfigBucketObjectName string
	JitaUsername                  string
	JitaPassword                  string
	JitaUrl                       string
	KolideEventHandlerAddress     string
	KolideEventHandlerToken       string
	KolideApiToken                string
}

type Azure struct {
	ClientID     string
	DiscoveryURL string
	ClientSecret string
}

func (c *Config) Credentials() (map[string]string, error) {
	credentials := make(map[string]string)
	for _, key := range c.CredentialEntries {
		entry := strings.Split(key, ":")
		if len(entry) > 2 {
			return nil, fmt.Errorf("invalid format on credentials, should be comma-separated entries on format 'user:key'")
		}

		credentials[entry[0]] = entry[1]
	}

	return credentials, nil
}

func DefaultConfig() Config {
	return Config{
		BindAddress:    "10.255.240.1:80",
		ConfigDir:      "/usr/local/etc/naisdevice/",
		PrometheusAddr: ":3000",
	}
}
