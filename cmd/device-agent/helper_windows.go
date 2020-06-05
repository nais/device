package main

import (
	"context"
	"os"
	"os/exec"

	"github.com/nais/device/device-agent/runtimeconfig"
)

func runHelper(rc *runtimeconfig.RuntimeConfig, ctx context.Context) error {
	cmd := adminCommandContext(ctx, "device-agent-helper.exe",
		"--interface", rc.Config.Interface,
		"--wireguard-binary", rc.Config.WireGuardBinary,
		"--wireguard-config-path", rc.Config.WireGuardConfigPath,
		"--log-level", rc.Config.LogLevel,
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Start()
}
func adminCommandContext(ctx context.Context, command string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "cmd", append([]string{"/c", "start", command}, arg...)...)
}
