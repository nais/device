package config

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/nais/device/internal/auth"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/wireguard"
	"github.com/sirupsen/logrus"
)

var MaxTenantId uint16 = (1 << 16) - 1

type Config struct {
	AutoEnrollEnabled                 bool
	Azure                             *auth.Azure
	BindAddress                       string
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
	KolideIntegrationEnabled          bool
	KolideApiToken                    string
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
	WireGuardIP                       string // for passing in raw string
	WireGuardIPv6                     string // for passing in raw string
	WireGuardConfigPath               string
	WireGuardPrivateKey               wireguard.PrivateKey
	WireGuardPrivateKeyPath           string
	WireGuardNetworkAddress           string
	WireGuardIPv4Prefix               *netip.Prefix `ignored:"true"`
	WireGuardIPv6Prefix               *netip.Prefix `ignored:"true"`
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
		WireGuardIP:                   "10.255.240.1",
		WireGuardConfigPath:           "/run/wg0.conf",
		WireGuardPrivateKeyPath:       "/etc/apiserver/private.key",
		GatewayConfigurer:             "bucket",
	}
}

func parsePrefixOrIP(raw string) (*netip.Prefix, error) {
	if strings.Contains(raw, "/") {
		p, err := netip.ParsePrefix(raw)
		if err != nil {
			return nil, err
		}
		return &p, nil
	} else {
		a, err := netip.ParseAddr(raw)
		if err != nil {
			return nil, err
		}
		var p netip.Prefix
		if a.Is6() {
			p = netip.PrefixFrom(a, 64)
		} else {
			p = netip.PrefixFrom(a, 21)
		}
		return &p, nil
	}
}

func (cfg *Config) Parse() error {
	var err error

	cfg.WireGuardIPv4Prefix, err = parsePrefixOrIP(cfg.WireGuardIP)
	if err != nil {
		return err
	}

	if len(cfg.WireGuardIPv6) != 0 {
		cfg.WireGuardIPv6Prefix, err = parsePrefixOrIP(cfg.WireGuardIPv6)
		if err != nil {
			return err
		}
	}

	if len(cfg.LogLevel) != 0 {
		level, err := logrus.ParseLevel(cfg.LogLevel)
		if err != nil {
			return err
		}
		cfg.LogLevel = level.String()
	}

	return nil
}

func (cfg *Config) APIServerPeer() *pb.Gateway {
	ipv6 := ""
	if cfg.WireGuardIPv6Prefix != nil {
		ipv6 = cfg.WireGuardIPv6Prefix.Addr().String()
	}

	return &pb.Gateway{
		Name:      "apiserver",
		PublicKey: string(cfg.WireGuardPrivateKey.Public()),
		Endpoint:  cfg.Endpoint,
		Ipv4:      cfg.WireGuardIPv4Prefix.Addr().String(),
		Ipv6:      ipv6,
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
