package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func runHelper(ctx context.Context, cfg Config) error {
	cmd := adminCommandContext(ctx, "./bin/device-agent-helper.exe",
		"--interface", cfg.Interface,
		"--wireguard-binary", cfg.WireGuardPath,
		"--wireguard-config-path", cfg.WireGuardConfigPath)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Start()
}

func GenerateBaseConfig(bootstrapConfig *BootstrapConfig, privateKey []byte) string {
	template := `[Interface]
PrivateKey = %s
MTU = 1360
Address = %s

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`
	return fmt.Sprintf(template, privateKey, bootstrapConfig.TunnelIP, bootstrapConfig.PublicKey, bootstrapConfig.APIServerIP, bootstrapConfig.Endpoint)
}

func setPlatformDefaults(cfg *Config) {
	programFiles := os.Getenv("%programfiles%")
	if programFiles == "" {
		programFiles = `c:\Program Files`
	}
	cfg.WireGuardPath = filepath.Join(programFiles, "WireGuard", "wireguard.exe")
}

func configDir() {
	filepath.Join()
}

func platformPrerequisites(cfg Config) error {
	return nil
}

func setPlatform(cfg *Config) {
	cfg.Platform = "windows"
}
