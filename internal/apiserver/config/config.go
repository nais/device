package config

import (
	"fmt"
	"net/netip"
	"strings"

	"github.com/nais/device/internal/token"
	"github.com/nais/device/internal/token/azure"
	"github.com/nais/device/internal/token/google"
	"github.com/nais/device/internal/wireguard"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
)

type Config struct {
	AutoEnrollEnabled                 bool
	AutoEnrollmentsURL                string
	Azure                             token.Config
	JITA                              token.Config
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
	Google                            token.Config
	KolideIntegrationEnabled          bool
	KolideAPIToken                    string
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
		Azure:                         azure.APIServerConfig,
		JITA:                          azure.JITAConfig,
		Google:                        google.APIServerConfig,
		BindAddress:                   "127.0.0.1:8080",
		DBPath:                        "/tmp/naisdevice.db",
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
