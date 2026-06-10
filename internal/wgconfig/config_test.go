package wgconfig_test

import (
	"net"
	"testing"
	"time"

	"github.com/nais/device/internal/wgconfig"
	"github.com/nais/device/internal/wireguard"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var testPrivateKey string

func init() {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		panic("generate test private key: " + err.Error())
	}
	testPrivateKey = key.String()
}

func generateTestPublicKey() string {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		panic("generate test key: " + err.Error())
	}
	return key.PublicKey().String()
}

func TestBuildConfig_BasicConversion(t *testing.T) {
	gwPubKey := generateTestPublicKey()

	cfg := &pb.Configuration{
		PrivateKey: testPrivateKey,
		DeviceIPv4: "10.255.24.1",
		DeviceIPv6: "fd00::1",
		Gateways: []*pb.Gateway{
			{
				Name:       "test-gw",
				PublicKey:  gwPubKey,
				Endpoint:   "1.2.3.4:51820",
				Ipv4:       "10.255.24.10",
				Ipv6:       "fd00::10",
				RoutesIPv4: []string{"10.0.0.0/24", "172.16.0.0/16"},
				RoutesIPv6: []string{"fd01::/64"},
			},
		},
	}

	wgCfg, err := wgconfig.BuildConfig(cfg)
	require.NoError(t, err)

	expectedPrivKey, _ := wgtypes.ParseKey(testPrivateKey)
	assert.Equal(t, expectedPrivKey, *wgCfg.PrivateKey)
	assert.True(t, wgCfg.ReplacePeers)
	require.Len(t, wgCfg.Peers, 1)

	peer := wgCfg.Peers[0]

	expectedPubKey, _ := wgtypes.ParseKey(gwPubKey)
	assert.Equal(t, expectedPubKey, peer.PublicKey)

	assert.Equal(t, "1.2.3.4", peer.Endpoint.IP.String())
	assert.Equal(t, 51820, peer.Endpoint.Port)

	expectedCIDRs := []string{
		"10.0.0.0/24",
		"172.16.0.0/16",
		"10.255.24.10/32",
		"fd00::10/128",
		"fd01::/64",
	}
	require.Len(t, peer.AllowedIPs, len(expectedCIDRs))
	for i, cidr := range expectedCIDRs {
		_, expected, _ := net.ParseCIDR(cidr)
		assert.Equal(t, *expected, peer.AllowedIPs[i], "AllowedIP mismatch at index %d", i)
	}

	assert.Nil(t, peer.PersistentKeepaliveInterval)
	assert.True(t, peer.ReplaceAllowedIPs)
}

func TestBuildConfig_PrometheusPeerKeepalive(t *testing.T) {
	promPubKey := generateTestPublicKey()

	cfg := &pb.Configuration{
		PrivateKey: testPrivateKey,
		Gateways: []*pb.Gateway{
			{
				Name:      wireguard.PrometheusPeerName,
				PublicKey: promPubKey,
				Endpoint:  "5.6.7.8:51820",
				Ipv4:      "10.255.24.20",
			},
		},
	}

	wgCfg, err := wgconfig.BuildConfig(cfg)
	require.NoError(t, err)

	require.Len(t, wgCfg.Peers, 1)
	peer := wgCfg.Peers[0]

	require.NotNil(t, peer.PersistentKeepaliveInterval)
	assert.Equal(t, 25*time.Second, *peer.PersistentKeepaliveInterval)
}

func TestBuildConfig_MultiplePeers(t *testing.T) {
	cfg := &pb.Configuration{
		PrivateKey: testPrivateKey,
		Gateways: []*pb.Gateway{
			{
				Name:      "gw-1",
				PublicKey: generateTestPublicKey(),
				Endpoint:  "1.1.1.1:51820",
				Ipv4:      "10.255.24.10",
			},
			{
				Name:      "gw-2",
				PublicKey: generateTestPublicKey(),
				Endpoint:  "2.2.2.2:51820",
				Ipv4:      "10.255.24.11",
			},
			{
				Name:      "gw-3",
				PublicKey: generateTestPublicKey(),
				Endpoint:  "3.3.3.3:51820",
				Ipv4:      "10.255.24.12",
			},
		},
	}

	wgCfg, err := wgconfig.BuildConfig(cfg)
	require.NoError(t, err)
	assert.Len(t, wgCfg.Peers, 3)
}

func TestBuildConfig_PeerWithoutEndpoint(t *testing.T) {
	cfg := &pb.Configuration{
		PrivateKey: testPrivateKey,
		Gateways: []*pb.Gateway{
			{
				Name:      "no-endpoint",
				PublicKey: generateTestPublicKey(),
				Ipv4:      "10.255.24.10",
			},
		},
	}

	wgCfg, err := wgconfig.BuildConfig(cfg)
	require.NoError(t, err)
	require.Len(t, wgCfg.Peers, 1)
	assert.Nil(t, wgCfg.Peers[0].Endpoint)
}

func TestBuildConfig_BareIPWithoutCIDRPrefix(t *testing.T) {
	cfg := &pb.Configuration{
		PrivateKey: testPrivateKey,
		Gateways: []*pb.Gateway{
			{
				Name:       "gw-bare-ip",
				PublicKey:  generateTestPublicKey(),
				Endpoint:   "1.2.3.4:51820",
				Ipv4:       "10.255.24.10",
				Ipv6:       "fd00::10",
				RoutesIPv4: []string{"10.43.0.60", "10.0.0.0/24"},
				RoutesIPv6: []string{"fd00::1"},
			},
		},
	}

	wgCfg, err := wgconfig.BuildConfig(cfg)
	require.NoError(t, err)
	require.Len(t, wgCfg.Peers, 1)

	peer := wgCfg.Peers[0]
	expectedCIDRs := []string{
		"10.43.0.60/32",
		"10.0.0.0/24",
		"10.255.24.10/32",
		"fd00::10/128",
		"fd00::1/128",
	}
	require.Len(t, peer.AllowedIPs, len(expectedCIDRs))
	for i, cidr := range expectedCIDRs {
		_, expected, _ := net.ParseCIDR(cidr)
		assert.Equal(t, *expected, peer.AllowedIPs[i], "AllowedIP mismatch at index %d", i)
	}
}

func TestBuildConfig_InvalidPrivateKey(t *testing.T) {
	cfg := &pb.Configuration{
		PrivateKey: "not-a-valid-key",
	}

	_, err := wgconfig.BuildConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse private key")
}

func TestBuildConfig_InvalidPeerPublicKey(t *testing.T) {
	cfg := &pb.Configuration{
		PrivateKey: testPrivateKey,
		Gateways: []*pb.Gateway{
			{
				Name:      "bad-key",
				PublicKey: "invalid",
				Ipv4:      "10.0.0.1",
			},
		},
	}

	_, err := wgconfig.BuildConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse public key")
}

func TestBuildConfig_NoGateways(t *testing.T) {
	cfg := &pb.Configuration{
		PrivateKey: testPrivateKey,
	}

	wgCfg, err := wgconfig.BuildConfig(cfg)
	require.NoError(t, err)
	assert.Empty(t, wgCfg.Peers)
	assert.True(t, wgCfg.ReplacePeers)
}
