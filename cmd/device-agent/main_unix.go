// +build linux darwin

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func runHelper(ctx context.Context, cfg Config) error {
	cmd := adminCommandContext(ctx, "./bin/device-agent-helper",
		"--interface", cfg.Interface,
		"--tunnel-ip", cfg.BootstrapConfig.TunnelIP,
		"--wireguard-binary", cfg.WireGuardPath,
		"--wireguard-go-binary", cfg.WireGuardGoBinary,
		"--wireguard-config-path", cfg.WireGuardConfigPath)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Start()
}

func GenerateBaseConfig(bootstrapConfig *BootstrapConfig, privateKey []byte) string {
	template := `[Interface]
PrivateKey = %s

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`
	return fmt.Sprintf(template, privateKey, bootstrapConfig.PublicKey, bootstrapConfig.APIServerIP, bootstrapConfig.Endpoint)
}

func setPlatformDefaults(cfg *Config) {
	cfg.WireGuardPath = filepath.Join(cfg.BinaryDir, "naisdevice-wg")
	cfg.WireGuardGoBinary = filepath.Join(cfg.BinaryDir, "naisdevice-wireguard-go")
}

func platformPrerequisites(cfg Config) error {
	if err := ensureDirectories(cfg.BinaryDir); err != nil {
		return fmt.Errorf("ensuring directory exists: %w", err)
	}

	return nil
}
