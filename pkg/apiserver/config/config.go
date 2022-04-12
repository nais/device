package config

import (
	"fmt"
	"strings"

	"github.com/nais/device/pkg/auth"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/wireguard"
)

type Config struct {
	AutoEnrollEnabled                 bool
	Azure                             *auth.Azure
	BindAddress                       string
	BootstrapAPIURL                   string
	BootstrapApiCredentials           string
	CloudSQLProxyEnabled              bool
	ControlPlaneAuthenticationEnabled bool
	AdminCredentialEntries            []string
	PrometheusCredentialEntries       []string
	DbConnDSN                         string
	DeviceAuthenticationProvider      string
	Endpoint                          string
	GRPCBindAddress                   string
	GatewayConfigBucketName           string
	GatewayConfigBucketObjectName     string
	Google                            *auth.Google
	JitaPassword                      string
	JitaUrl                           string
	JitaUsername                      string
	JitaEnabled                       bool
	KolideEventHandlerAddress         string
	KolideEventHandlerEnabled         bool
	KolideEventHandlerToken           string
	KolideEventHandlerSecure          bool
	LogLevel                          string
	PrometheusAddr                    string
	PrometheusPublicKey               string
	PrometheusTunnelIP                string
	GatewayConfigurer                 string
	WireGuardEnabled                  bool
	WireGuardIP                       string
	WireGuardConfigPath               string
	WireGuardPrivateKey               wireguard.PrivateKey
	WireGuardPrivateKeyPath           string
	WireGuardNetworkAddress           string
}

func Credentials(entries []string) (map[string]string, error) {
	credentials := make(map[string]string)
	for _, key := range entries {
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
		Azure: &auth.Azure{
			ClientID: "6e45010d-2637-4a40-b91d-d4cbb451fb57",
			Tenant:   "62366534-1ec3-4962-8869-9b5535279d0b",
		},
		Google: &auth.Google{
			ClientID: "955023559628-g51n36t4icbd6lq7ils4r0ol9oo8kpk0.apps.googleusercontent.com",
		},
		BindAddress:                   "127.0.0.1:8080",
		DbConnDSN:                     "postgresql://postgres:postgres@localhost/postgres?sslmode=disable",
		GRPCBindAddress:               "127.0.0.1:8099",
		GatewayConfigBucketName:       "gatewayconfig",
		GatewayConfigBucketObjectName: "gatewayconfig.json",
		LogLevel:                      "info",
		PrometheusAddr:                "127.0.0.1:3000",
		WireGuardNetworkAddress:       "10.255.240.0/21",
		WireGuardIP:                   "10.255.240.1",
		WireGuardConfigPath:           "/run/wg0.conf",
		WireGuardPrivateKeyPath:       "/etc/apiserver/private.key",
		GatewayConfigurer:             "bucket",
	}
}

func (cfg *Config) APIServerPeer() *pb.Gateway {
	return &pb.Gateway{
		Name:      "apiserver",
		PublicKey: string(cfg.WireGuardPrivateKey.Public()),
		Endpoint:  cfg.Endpoint,
		Ip:        cfg.WireGuardIP,
	}
}

func (cfg *Config) StaticPeers() []*pb.Gateway {
	return []*pb.Gateway{
		{
			Name:      "prometheus",
			PublicKey: cfg.PrometheusPublicKey,
			Ip:        cfg.PrometheusTunnelIP,
		},
	}
}
