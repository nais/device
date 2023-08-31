package config

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/nais/device/pkg/auth"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/wireguard"
)

const wireGuardV6PrefixAddress = "fd75:568f:0d24::/48"

type Config struct {
	AutoEnrollEnabled                 bool
	Azure                             *auth.Azure
	BindAddress                       string
	BootstrapAPIURL                   string
	BootstrapApiCredentials           string
	ControlPlaneAuthenticationEnabled bool
	AdminCredentialEntries            []string
	PrometheusCredentialEntries       []string
	DBPath                            string
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
	WireGuardIPv4                     string
	WireGuardIPv6                     netip.Prefix
	WireGuardConfigPath               string
	WireGuardPrivateKey               wireguard.PrivateKey
	WireGuardPrivateKeyPath           string
	WireGuardNetworkAddress           string
	TenantID                          uint16
}

// Generate a unique IPv6 /64 address for a tenant, placing the tenant id as the 7th and 8th bytes of the IPv6 prefix.
func getWireGuardIPv6(tenantId uint16) netip.Prefix {
	b := netip.MustParsePrefix(wireGuardV6PrefixAddress).Addr().As16()
	b[6] = byte(tenantId >> 8)
	b[7] = byte(tenantId)
	return netip.PrefixFrom(netip.AddrFrom16(b), 64)
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
		DBPath:                        "sqlite3:///tmp/naisdevice.db",
		GRPCBindAddress:               "127.0.0.1:8099",
		GatewayConfigBucketName:       "gatewayconfig",
		GatewayConfigBucketObjectName: "gatewayconfig.json",
		LogLevel:                      "info",
		PrometheusAddr:                "127.0.0.1:3000",
		WireGuardNetworkAddress:       "10.255.240.0/21",
		WireGuardIPv4:                 "10.255.240.1",
		WireGuardIPv6:                 getWireGuardIPv6(0),
		WireGuardConfigPath:           "/run/wg0.conf",
		WireGuardPrivateKeyPath:       "/etc/apiserver/private.key",
		GatewayConfigurer:             "bucket",
	}
}

func (cfg *Config) Parse() {
	cfg.WireGuardIPv6 = getWireGuardIPv6(cfg.TenantID)
}

func (cfg *Config) APIServerPeer() *pb.Gateway {
	return &pb.Gateway{
		Name:      "apiserver",
		PublicKey: string(cfg.WireGuardPrivateKey.Public()),
		Endpoint:  cfg.Endpoint,
		Ipv4:      cfg.WireGuardIPv4,
		Ipv6:      cfg.WireGuardIPv6.Addr().StringExpanded(),
	}
}

func (cfg *Config) StaticPeers() []*pb.Gateway {
	return []*pb.Gateway{
		{
			Name:      wireguard.PrometheusPeerName,
			PublicKey: cfg.PrometheusPublicKey,
			Ipv4:      cfg.PrometheusTunnelIP,
		},
	}
}
