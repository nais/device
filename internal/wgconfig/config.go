// Package wgconfig converts protobuf Configuration types to wgtypes.Config
// and applies them via wgctrl.Client.ConfigureDevice().
package wgconfig

import (
	"context"
	"fmt"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/nais/device/internal/wireguard"
	"github.com/nais/device/pkg/pb"
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

	return wireguard.DeviceConfig(privateKey, nil, wireguard.CastPeerList(cfg.GetGateways()), current)
}
