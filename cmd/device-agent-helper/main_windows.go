package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

const (
	WireGuardBinary = `c:\Program Files\WireGuard\wireguard.exe`
)

func platformFlags(cfg *Config) {}

func setupInterface(ctx context.Context, cfg Config) error {
	teardownInterface(ctx, cfg)

	installService := exec.CommandContext(ctx, cfg.WireGuardBinary, "/installtunnelservice", cfg.WireGuardConfigPath)
	if b, err := installService.CombinedOutput(); err != nil {
		return fmt.Errorf("installing tunnel service: %v: %v", err, string(b))
	} else {
		log.Infof("installed tunnel service: %v", string(b))
	}

	return nil
}

func syncConf(cfg Config, ctx context.Context) error {
	commands := [][]string{
		{"net", "stop", serviceName(cfg.Interface)},
		{"net", "start", serviceName(cfg.Interface)},
	}

	return runCommands(ctx, commands)
}

func teardownInterface(ctx context.Context, cfg Config) {
	queryService := exec.CommandContext(ctx, "sc", "query", serviceName(cfg.Interface))
	if b, err := queryService.CombinedOutput(); err != nil {
		log.Infof("no previously installed service detected: %v: %v", err, string(b))
	} else {
		uninstallService := exec.CommandContext(ctx, cfg.WireGuardBinary, "/uninstalltunnelservice", cfg.Interface)
		if b, err := uninstallService.CombinedOutput(); err != nil {
			log.Infof("uninstalling tunnel service, this is ok: %v", err)
		} else {
			log.Infof("uninstalled tunnel service: %v", string(b))
		}
	}
}

func prerequisites() error {
	if err := filesExist(cfg.WireGuardBinary); err != nil {
		return fmt.Errorf("Verifying if file exists: %w", err)
	}

	return nil
}

func serviceName(interfaceName string) string {
	return fmt.Sprintf("WireGuardTunnel$%s", interfaceName)
}
