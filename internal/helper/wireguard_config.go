package helper

import (
	"fmt"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/nais/device/internal/wireguard"
	"github.com/nais/device/pkg/pb"
)

func applyWireGuardConfig(iface string, cfg *pb.Configuration) error {
	privateKey, err := wgtypes.ParseKey(cfg.GetPrivateKey())
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}

	return wireguard.ConfigureDevice(iface, privateKey, nil, wireguard.CastPeerList(cfg.GetGateways()))
}
