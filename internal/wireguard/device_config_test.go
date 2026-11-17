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

func TestDeviceConfig(t *testing.T) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)
	listenPort := 51820
	kept, added, removed := newKey(t), newKey(t), newKey(t)

	peers := []wireguard.Peer{
		&pb.Gateway{Name: "kept", PublicKey: kept.String(), Ipv4: "10.255.240.2", Endpoint: "127.0.0.1:51820"},
		&pb.Gateway{Name: wireguard.PrometheusPeerName, PublicKey: added.String(), Ipv4: "10.255.240.3"},
	}
	current := []wgtypes.Peer{{PublicKey: kept}, {PublicKey: removed}}

	cfg, err := wireguard.DeviceConfig(privateKey, &listenPort, peers, current)
	require.NoError(t, err)

	assert.Equal(t, privateKey, *cfg.PrivateKey)
	assert.Equal(t, 51820, *cfg.ListenPort)
	assert.False(t, cfg.ReplacePeers, "peers must be updated in place to keep sessions")
	require.Len(t, cfg.Peers, 3)

	assert.Equal(t, kept, cfg.Peers[0].PublicKey)
	assert.True(t, cfg.Peers[0].ReplaceAllowedIPs)
	assert.Equal(t, []net.IPNet{{IP: net.IPv4(10, 255, 240, 2).To4(), Mask: net.CIDRMask(32, 32)}}, cfg.Peers[0].AllowedIPs)
	assert.Equal(t, "127.0.0.1:51820", cfg.Peers[0].Endpoint.String())
	assert.Nil(t, cfg.Peers[0].PersistentKeepaliveInterval)

	assert.Equal(t, added, cfg.Peers[1].PublicKey)
	assert.Nil(t, cfg.Peers[1].Endpoint)
	require.NotNil(t, cfg.Peers[1].PersistentKeepaliveInterval)
	assert.Equal(t, 25*time.Second, *cfg.Peers[1].PersistentKeepaliveInterval)

	assert.Equal(t, wgtypes.PeerConfig{PublicKey: removed, Remove: true}, cfg.Peers[2])
}

func TestDeviceConfig_NilListenPortLeavesPortUnchanged(t *testing.T) {
	cfg, err := wireguard.DeviceConfig(wgtypes.Key{}, nil, nil, nil)
	require.NoError(t, err)
	assert.Nil(t, cfg.ListenPort)
	assert.Empty(t, cfg.Peers)
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
