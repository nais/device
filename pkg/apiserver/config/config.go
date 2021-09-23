package config

import (
	"fmt"
	"strings"
)

const NaisDeviceApprovalGroup = "ffd89425-c75c-4618-b5ab-67149ddbbc2d"

type Config struct {
	Azure                             Azure
	BindAddress                       string
	BootstrapAPIURL                   string
	BootstrapApiCredentials           string
	CloudSQLProxyEnabled              bool
	ConfigDir                         string
	ControlPlaneAuthenticationEnabled bool
	CredentialEntries                 []string
	DbConnDSN                         string
	DeviceAuthenticationEnabled       bool
	Endpoint                          string
	GRPCBindAddress                   string
	GatewayConfigBucketName           string
	GatewayConfigBucketObjectName     string
	JitaPassword                      string
	JitaUrl                           string
	JitaUsername                      string
	KolideApiToken                    string
	KolideEventHandlerAddress         string
	KolideEventHandlerEnabled         bool
	KolideEventHandlerToken           string
	KolideSyncEnabled                 bool
	LogLevel                          string
	PrivateKeyPath                    string
	PrometheusAddr                    string
	PrometheusPublicKey               string
	PrometheusTunnelIP                string
	WireGuardConfigPath               string
	WireguardEnabled                  bool
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

func (c *Config) DatabaseDriver() string {
	if c.CloudSQLProxyEnabled {
		return "cloudsqlpostgres"
	}
	return "postgres"
}

func DefaultConfig() Config {
	return Config{
		BindAddress:     "127.0.0.1:8080",
		GRPCBindAddress: "127.0.0.1:8099",
		ConfigDir:       "/usr/local/etc/naisdevice/",
		PrometheusAddr:  "127.0.0.1:3000",
	}
}
