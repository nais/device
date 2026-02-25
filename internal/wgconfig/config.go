// Package wgconfig converts protobuf Configuration types to wgtypes.Config
// and applies them via wgctrl.Client.ConfigureDevice().
package wgconfig

import (
	"fmt"
	"net"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/nais/device/internal/wireguard"
	"github.com/nais/device/pkg/pb"
)

const (
	defaultPersistentKeepalive = 25 * time.Second
)

// ApplyConfig configures the named WireGuard device with the given configuration.
// It replaces all existing peers.
func ApplyConfig(ifaceName string, cfg *pb.Configuration) error {
	wgCfg, err := BuildConfig(cfg)
	if err != nil {
		return fmt.Errorf("build wgctrl config: %w", err)
	}

	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("create wgctrl client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if err := client.ConfigureDevice(ifaceName, *wgCfg); err != nil {
		return fmt.Errorf("configure device %q: %w", ifaceName, err)
	}

	return nil
}

// BuildConfig converts a protobuf Configuration into a wgtypes.Config.
func BuildConfig(cfg *pb.Configuration) (*wgtypes.Config, error) {
	privateKey, err := wgtypes.ParseKey(cfg.GetPrivateKey())
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	gateways := cfg.GetGateways()
	peers := make([]wgtypes.PeerConfig, 0, len(gateways))
	for _, gw := range gateways {
		peerCfg, err := buildPeerConfig(gw)
		if err != nil {
			return nil, fmt.Errorf("build peer config for %q: %w", gw.GetName(), err)
		}
		peers = append(peers, *peerCfg)
	}

	replacePeers := true
	return &wgtypes.Config{
		PrivateKey:   &privateKey,
		ReplacePeers: replacePeers,
		Peers:        peers,
	}, nil
}

// BuildServerConfig converts a wireguard.Config into a wgtypes.Config.
func BuildServerConfig(cfg *wireguard.Config) (*wgtypes.Config, error) {
	privateKey, err := wgtypes.ParseKey(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	peers := make([]wgtypes.PeerConfig, 0, len(cfg.Peers))
	for _, peer := range cfg.Peers {
		peerCfg, err := buildPeerConfig(peer)
		if err != nil {
			return nil, fmt.Errorf("build peer config for %q: %w", peer.GetName(), err)
		}
		peers = append(peers, *peerCfg)
	}

	listenPort := cfg.ListenPort

	replacePeers := true
	wgCfg := &wgtypes.Config{
		PrivateKey:   &privateKey,
		ReplacePeers: replacePeers,
		Peers:        peers,
	}

	if listenPort > 0 {
		wgCfg.ListenPort = &listenPort
	}

	return wgCfg, nil
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
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("parse CIDR %q: %w", cidr, err)
		}
		nets = append(nets, *ipNet)
	}
	return nets, nil
}
