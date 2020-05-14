// +build linux darwin

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nais/device/device-agent/config"
)

func runHelper(ctx context.Context, cfg config.Config) error {
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

func GenerateBaseConfig(bootstrapConfig *config.BootstrapConfig, privateKey []byte) string {
	template := `[Interface]
PrivateKey = %s

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`
	return fmt.Sprintf(template, KeyToBase64(privateKey), bootstrapConfig.PublicKey, bootstrapConfig.APIServerIP, bootstrapConfig.Endpoint)
}

func setPlatformDefaults(cfg *config.Config) {
	cfg.WireGuardBinary = filepath.Join(cfg.BinaryDir, "naisdevice-wg")
	cfg.WireGuardGoBinary = filepath.Join(cfg.BinaryDir, "naisdevice-wireguard-go")
}

func platformPrerequisites(cfg config.Config) error {
	if err := ensureDirectories(cfg.BinaryDir); err != nil {
		return fmt.Errorf("ensuring directory exists: %w", err)
	}

	if err := filesExist(cfg.WireGuardGoBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}

	return nil
}
