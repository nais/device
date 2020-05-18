// +build linux darwin

package main

import (
	"context"
	"os"

	"github.com/nais/device/device-agent/config"
)

func runHelper(cfg *config.Config, ctx context.Context) error {
	cmd := adminCommandContext(ctx, "./bin/device-agent-helper",
		"--interface", cfg.Interface,
		"--tunnel-ip", cfg.BootstrapConfig.TunnelIP,
		"--wireguard-binary", cfg.WireGuardBinary,

		"--wireguard-go-binary", cfg.WireGuardGoBinary,
		"--wireguard-config-path", cfg.WireGuardConfigPath)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Start()
}
