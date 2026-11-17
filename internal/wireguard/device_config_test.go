package wireguard_test

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/nais/device/internal/wireguard"
	"github.com/nais/device/pkg/pb"
)

func newKey(t *testing.T) wgtypes.Key {
	t.Helper()
	key, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)
	return key.PublicKey()
}

func cidrs(t *testing.T, raw ...string) []net.IPNet {
	t.Helper()
	nets := make([]net.IPNet, len(raw))
	for i, cidr := range raw {
		_, ipnet, err := net.ParseCIDR(cidr)
		require.NoError(t, err)
		nets[i] = *ipnet
	}
	return nets
}

func udpAddr(t *testing.T, raw string) *net.UDPAddr {
	t.Helper()
	addr, err := net.ResolveUDPAddr("udp", raw)
	require.NoError(t, err)
	return addr
}

func TestDeviceConfig(t *testing.T) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)
	listenPort := 51820
	keepalive := 25 * time.Second
	gw, prometheus, other, stale := newKey(t), newKey(t), newKey(t), newKey(t)

	tests := []struct {
		name       string
		listenPort *int
		peers      []wireguard.Peer
		current    []wgtypes.Peer
		want       []wgtypes.PeerConfig
	}{
		{
			name: "no peers",
			want: []wgtypes.PeerConfig{},
		},
		{
			name:       "listen port is set",
			listenPort: &listenPort,
			want:       []wgtypes.PeerConfig{},
		},
		{
			name: "gateway with routes and endpoint",
			peers: []wireguard.Peer{&pb.Gateway{
				Name:       "gw",
				PublicKey:  gw.String(),
				Endpoint:   "1.2.3.4:51820",
				Ipv4:       "10.255.24.10",
				Ipv6:       "fd00::10",
				RoutesIPv4: []string{"10.0.0.0/24", "172.16.0.0/16"},
				RoutesIPv6: []string{"fd01::/64"},
			}},
			want: []wgtypes.PeerConfig{{
				PublicKey:         gw,
				Endpoint:          udpAddr(t, "1.2.3.4:51820"),
				ReplaceAllowedIPs: true,
				AllowedIPs:        cidrs(t, "10.0.0.0/24", "172.16.0.0/16", "10.255.24.10/32", "fd00::10/128", "fd01::/64"),
			}},
		},
		{
			name: "bare route IPs become host routes",
			peers: []wireguard.Peer{&pb.Gateway{
				Name:       "gw",
				PublicKey:  gw.String(),
				Ipv4:       "10.255.24.10",
				Ipv6:       "fd00::10",
				RoutesIPv4: []string{"10.43.0.60"},
				RoutesIPv6: []string{"fd00::1"},
			}},
			want: []wgtypes.PeerConfig{{
				PublicKey:         gw,
				ReplaceAllowedIPs: true,
				AllowedIPs:        cidrs(t, "10.43.0.60/32", "10.255.24.10/32", "fd00::10/128", "fd00::1/128"),
			}},
		},
		{
			name:  "prometheus peer gets keepalive",
			peers: []wireguard.Peer{&pb.Gateway{Name: wireguard.PrometheusPeerName, PublicKey: prometheus.String(), Ipv4: "10.255.24.20"}},
			want: []wgtypes.PeerConfig{{
				PublicKey:                   prometheus,
				ReplaceAllowedIPs:           true,
				AllowedIPs:                  cidrs(t, "10.255.24.20/32"),
				PersistentKeepaliveInterval: &keepalive,
			}},
		},
		{
			name: "current peers are kept in place or removed",
			peers: []wireguard.Peer{
				&pb.Gateway{Name: "gw", PublicKey: gw.String(), Ipv4: "10.255.24.10"},
				&pb.Gateway{Name: "other", PublicKey: other.String(), Ipv4: "10.255.24.11"},
			},
			current: []wgtypes.Peer{{PublicKey: gw}, {PublicKey: stale}},
			want: []wgtypes.PeerConfig{
				{PublicKey: gw, ReplaceAllowedIPs: true, AllowedIPs: cidrs(t, "10.255.24.10/32")},
				{PublicKey: other, ReplaceAllowedIPs: true, AllowedIPs: cidrs(t, "10.255.24.11/32")},
				{PublicKey: stale, Remove: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := wireguard.DeviceConfig(privateKey, tt.listenPort, tt.peers, tt.current)
			require.NoError(t, err)

			assert.Equal(t, privateKey, *cfg.PrivateKey)
			assert.Equal(t, tt.listenPort, cfg.ListenPort)
			assert.False(t, cfg.ReplacePeers, "peers must be updated in place to keep sessions")
			assert.Equal(t, tt.want, cfg.Peers)
		})
	}
}

func TestDeviceConfig_InvalidPeers(t *testing.T) {
	raw := newKey(t)
	tests := []struct {
		name    string
		peer    *pb.Gateway
		wantErr string
	}{
		{
			name:    "raw 32-byte public key",
			peer:    &pb.Gateway{Name: "raw", PublicKey: string(raw[:]), Ipv4: "10.255.240.2"},
			wantErr: `build peer config for "raw": parse public key`,
		},
		{
			name:    "empty public key",
			peer:    &pb.Gateway{Name: "empty", Ipv4: "10.255.240.2"},
			wantErr: `build peer config for "empty": parse public key`,
		},
		{
			name:    "invalid allowed IP",
			peer:    &pb.Gateway{Name: "bad-ip", PublicKey: newKey(t).String(), Ipv4: "not-an-ip"},
			wantErr: `build peer config for "bad-ip": parse allowed IPs`,
		},
		{
			name:    "invalid endpoint",
			peer:    &pb.Gateway{Name: "bad-endpoint", PublicKey: newKey(t).String(), Ipv4: "10.255.240.2", Endpoint: "no-port"},
			wantErr: `build peer config for "bad-endpoint": resolve endpoint`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := wireguard.DeviceConfig(wgtypes.Key{}, nil, []wireguard.Peer{tt.peer}, nil)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
