// +build linux darwin

package main

import (
	"context"
	"github.com/nais/device/device-agent/runtimeconfig"
	"os"
	"os/exec"
)

func runHelper(rc *runtimeconfig.RuntimeConfig, ctx context.Context) error {
	cmd := adminCommandContext(ctx, "./bin/macos/device-agent-helper",
		"--interface", rc.Config.Interface,
		"--device-ip", rc.BootstrapConfig.DeviceIP,
		"--wireguard-binary", rc.Config.WireGuardBinary,
		"--wireguard-go-binary", rc.Config.WireGuardGoBinary,
		"--wireguard-config-path", rc.Config.WireGuardConfigPath,
		"--log-level", rc.Config.LogLevel,
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Start()
}

func adminCommandContext(ctx context.Context, command string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "sudo", append([]string{command}, arg...)...)
}
