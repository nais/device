package wireguard

import (
	"fmt"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/nais/device/internal/iputil"
)

const prometheusPersistentKeepalive = 25 * time.Second

// DeviceConfig returns a config that turns the current peers into the desired ones.
// Existing peers are updated in place, so their sessions survive the update.
// A nil listenPort leaves the device's listen port unchanged.
func DeviceConfig(privateKey wgtypes.Key, listenPort *int, peers []Peer, current []wgtypes.Peer) (*wgtypes.Config, error) {
	peerConfigs := make([]wgtypes.PeerConfig, 0, len(peers))
	desired := make(map[wgtypes.Key]bool, len(peers))
	for _, peer := range peers {
		peerCfg, err := peerConfig(peer)
		if err != nil {
			return nil, fmt.Errorf("build peer config for %q: %w", peer.GetName(), err)
		}
		peerConfigs = append(peerConfigs, *peerCfg)
		desired[peerCfg.PublicKey] = true
	}

	for _, peer := range current {
		if !desired[peer.PublicKey] {
			peerConfigs = append(peerConfigs, wgtypes.PeerConfig{PublicKey: peer.PublicKey, Remove: true})
		}
	}

	return &wgtypes.Config{
		PrivateKey: &privateKey,
		ListenPort: listenPort,
		Peers:      peerConfigs,
	}, nil
}

func peerConfig(peer Peer) (*wgtypes.PeerConfig, error) {
	publicKey, err := wgtypes.ParseKey(peer.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	allowedIPs, err := parseAllowedIPs(peer.GetAllowedIPs())
	if err != nil {
		return nil, fmt.Errorf("parse allowed IPs: %w", err)
	}

	peerCfg := &wgtypes.PeerConfig{
		PublicKey:         publicKey,
		ReplaceAllowedIPs: true,
		AllowedIPs:        allowedIPs,
	}

	if endpoint := peer.GetEndpoint(); endpoint != "" {
		addr, err := net.ResolveUDPAddr("udp", endpoint)
		if err != nil {
			return nil, fmt.Errorf("resolve endpoint %q: %w", endpoint, err)
		}
		peerCfg.Endpoint = addr
	}

	if peer.GetName() == PrometheusPeerName {
		keepalive := prometheusPersistentKeepalive
		peerCfg.PersistentKeepaliveInterval = &keepalive
	}

	return peerCfg, nil
}

func parseAllowedIPs(cidrs []string) ([]net.IPNet, error) {
	nets := make([]net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		prefix, err := iputil.ParsePrefix(cidr)
		if err != nil {
			return nil, fmt.Errorf("parse allowed IP: %w", err)
		}
		bits := 128
		if prefix.Addr().Is4() {
			bits = 32
		}
		nets = append(nets, net.IPNet{
			IP:   prefix.Addr().AsSlice(),
			Mask: net.CIDRMask(prefix.Bits(), bits),
		})
	}
	return nets, nil
}
