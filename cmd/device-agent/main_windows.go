package main

import (
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
