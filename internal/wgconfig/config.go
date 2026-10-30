// Package wgconfig converts protobuf Configuration types to wgtypes.Config
// and applies them via wgctrl.Client.ConfigureDevice().
package wgconfig

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/nais/device/internal/iputil"
	"github.com/nais/device/internal/wireguard"
	"github.com/nais/device/pkg/pb"
)

const (
	defaultPersistentKeepalive = 25 * time.Second
)

// ApplyConfig configures the named WireGuard device with the given configuration.
// Peers not in cfg are removed. Existing peers are updated in place, so their
// sessions survive the update.
func ApplyConfig(ctx context.Context, ifaceName string, cfg *pb.Configuration) error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("create wgctrl client: %w", err)
	}
	defer func() { _ = client.Close() }()

	device, err := client.Device(ifaceName)
	if err != nil {
		return fmt.Errorf("get device %q: %w", ifaceName, err)
	}

	wgCfg, err := BuildConfig(cfg, device.Peers)
	if err != nil {
		return fmt.Errorf("build wgctrl config: %w", err)
	}

	if err := client.ConfigureDevice(ifaceName, *wgCfg); err != nil {
		return fmt.Errorf("configure device %q: %w", ifaceName, err)
	}

	return nil
}

// BuildConfig converts a protobuf Configuration into a wgtypes.Config that
// turns the current peers into the desired ones without replacing all peers.
func BuildConfig(cfg *pb.Configuration, current []wgtypes.Peer) (*wgtypes.Config, error) {
	privateKey, err := wgtypes.ParseKey(cfg.GetPrivateKey())
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	gateways := cfg.GetGateways()
	peers := make([]wgtypes.PeerConfig, 0, len(gateways))
	desired := make(map[wgtypes.Key]bool, len(gateways))
	for _, gw := range gateways {
		peerCfg, err := buildPeerConfig(gw)
		if err != nil {
			return nil, fmt.Errorf("build peer config for %q: %w", gw.GetName(), err)
		}
		peers = append(peers, *peerCfg)
		desired[peerCfg.PublicKey] = true
	}

	for _, peer := range current {
		if !desired[peer.PublicKey] {
			peers = append(peers, wgtypes.PeerConfig{PublicKey: peer.PublicKey, Remove: true})
		}
	}

	return &wgtypes.Config{
		PrivateKey: &privateKey,
		Peers:      peers,
	}, nil
}

func buildPeerConfig(peer wireguard.Peer) (*wgtypes.PeerConfig, error) {
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

	if peer.GetName() == wireguard.PrometheusPeerName {
		keepalive := defaultPersistentKeepalive
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
		addr := prefix.Addr()
		ones := prefix.Bits()
		var bits int
		if addr.Is4() {
			bits = 32
		} else {
			bits = 128
		}
		nets = append(nets, net.IPNet{
			IP:   addr.AsSlice(),
			Mask: net.CIDRMask(ones, bits),
		})
	}
	return nets, nil
}
