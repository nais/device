package main

import (
	"context"
	"os"

	"github.com/nais/device/device-agent/config"
)

func runHelper(cfg *config.Config, ctx context.Context) error {
	cmd := adminCommandContext(ctx, "./bin/device-agent-helper.exe",
		"--interface", cfg.Interface,
		"--wireguard-binary", cfg.WireGuardBinary,
		"--wireguard-config-path", cfg.WireGuardConfigPath)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Start()
}
