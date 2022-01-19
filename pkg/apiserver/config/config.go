package config

import (
	"fmt"
	"strings"

	"github.com/nais/device/pkg/azure"
)

type Config struct {
	Azure                             *azure.Azure
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
		Azure: &azure.Azure{
			ClientID: "6e45010d-2637-4a40-b91d-d4cbb451fb57",
			Tenant:   "62366534-1ec3-4962-8869-9b5535279d0b",
		},
		BindAddress:                   "127.0.0.1:8080",
		ConfigDir:                     "/usr/local/etc/naisdevice/",
		DbConnDSN:                     "postgresql://postgres:postgres@localhost/postgres?sslmode=disable",
		GRPCBindAddress:               "127.0.0.1:8099",
		GatewayConfigBucketName:       "gatewayconfig",
		GatewayConfigBucketObjectName: "gatewayconfig.json",
		LogLevel:                      "info",
		PrometheusAddr:                "127.0.0.1:3000",
	}
}
