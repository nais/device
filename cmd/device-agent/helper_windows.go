package main

import (
	"context"
	"os"

	"github.com/nais/device/device-agent/runtimeconfig"
)

func runHelper(rc *runtimeconfig.RuntimeConfig, ctx context.Context) error {
	cmd := adminCommandContext(ctx, "./bin/device-agent-helper.exe",
		"--interface", rc.Config.Interface,
		"--wireguard-binary", rc.Config.WireGuardBinary,
		"--wireguard-config-path", rc.Config.WireGuardConfigPath)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Start()
}
