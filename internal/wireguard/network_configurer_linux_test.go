package wireguard_test

import (
	"net"
	"net/netip"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/nais/device/internal/wireguard"
	"github.com/nais/device/pkg/pb"
)

// Needs root and kernel WireGuard: NAISDEVICE_NETLINK_TEST=1 sudo -E go test ./internal/wireguard
func TestNetworkConfigurer(t *testing.T) {
	if os.Getenv("NAISDEVICE_NETLINK_TEST") == "" {
		t.Skip("set NAISDEVICE_NETLINK_TEST=1 and run as root to create a real WireGuard interface")
	}

	const iface = "nd-test0"
	const listenPort = 51999
	v4 := netip.MustParsePrefix("10.99.0.1/24")
	v6 := netip.MustParsePrefix("fd99::1/64")
	privateKey, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)

	t.Cleanup(func() {
		if link, err := netlink.LinkByName(iface); err == nil {
			_ = netlink.LinkDel(link)
		}
	})

	nc, err := wireguard.NewConfigurer(logrus.NewEntry(logrus.New()), &v4, &v6, privateKey.String(), iface, listenPort, nil, nil, nil)
	require.NoError(t, err)

	// Twice, as the agents recreate an interface left over from a previous run.
	for range 2 {
		require.NoError(t, nc.SetupInterface())

		link, err := netlink.LinkByName(iface)
		require.NoError(t, err)
		assert.Equal(t, "wireguard", link.Type())
		assert.Equal(t, 1360, link.Attrs().MTU)
		assert.NotZero(t, link.Attrs().Flags&net.FlagUp, "interface should be up")

		addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
		require.NoError(t, err)
		var got []string
		for _, addr := range addrs {
			if !addr.IP.IsLinkLocalUnicast() {
				got = append(got, addr.IPNet.String())
			}
		}
		assert.ElementsMatch(t, []string{"10.99.0.1/24", "fd99::1/64"}, got)
	}

	client, err := wgctrl.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	kept, removed := newKey(t), newKey(t)
	peers := []wireguard.Peer{
		&pb.Gateway{Name: "kept", PublicKey: kept.String(), Ipv4: "10.99.0.2", Endpoint: "127.0.0.1:51820"},
		&pb.Gateway{Name: "removed", PublicKey: removed.String(), Ipv4: "10.99.0.3"},
	}
	require.NoError(t, nc.ApplyWireGuardConfig(peers))

	device, err := client.Device(iface)
	require.NoError(t, err)
	assert.Equal(t, privateKey, device.PrivateKey)
	assert.Equal(t, listenPort, device.ListenPort)
	assert.ElementsMatch(t, []wgtypes.Key{kept, removed}, peerKeys(device.Peers))

	require.NoError(t, nc.ApplyWireGuardConfig(peers[:1]))

	device, err = client.Device(iface)
	require.NoError(t, err)
	require.Len(t, device.Peers, 1)
	assert.Equal(t, kept, device.Peers[0].PublicKey)
	assert.Equal(t, "10.99.0.2/32", device.Peers[0].AllowedIPs[0].String())
	assert.Equal(t, "127.0.0.1:51820", device.Peers[0].Endpoint.String())
}

func peerKeys(peers []wgtypes.Peer) []wgtypes.Key {
	keys := make([]wgtypes.Key, len(peers))
	for i, peer := range peers {
		keys[i] = peer.PublicKey
	}
	return keys
}
