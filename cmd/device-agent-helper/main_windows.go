package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	WireGuardBinary = `c:\Program Files\WireGuard\wireguard.exe`
)

func platformFlags(cfg *Config) {}

func setupInterface(ctx context.Context, cfg Config) error {
	teardownInterface(ctx, cfg)
	log.Info("Allowing Windows to process its existential crisis while tearing down any previous WG interface (aka sleep 5 sec)")
	time.Sleep(5 * time.Second)

	installService := exec.CommandContext(ctx, cfg.WireGuardBinary, "/installtunnelservice", cfg.WireGuardConfigPath)
	if b, err := installService.CombinedOutput(); err != nil {
		return fmt.Errorf("installing tunnel service: %v: %v", err, string(b))
	} else {
		log.Infof("installed tunnel service: %v", string(b))
	}

	return nil
}

var oldWireGuardConfig []byte

func syncConf(cfg Config, ctx context.Context) error {
	newWireGuardConfig, err := ioutil.ReadFile(cfg.WireGuardConfigPath)
	if err != nil {
		return fmt.Errorf("reading WireGuard config file: %w", err)
	}

	if fileActuallyChanged(oldWireGuardConfig, newWireGuardConfig) {
		oldWireGuardConfig = newWireGuardConfig

		commands := [][]string{
			{"net", "stop", serviceName(cfg.Interface)},
			{"net", "start", serviceName(cfg.Interface)},
		}

		return runCommands(ctx, commands)
	}

	return nil
}

func teardownInterface(ctx context.Context, cfg Config) {
	queryService := exec.CommandContext(ctx, "sc", "query", serviceName(cfg.Interface))
	if b, err := queryService.CombinedOutput(); err != nil {
		log.Infof("querying for existing service (probably not found, which is fine): %v: %v", err, string(b))
	} else {
		uninstallService := exec.CommandContext(ctx, cfg.WireGuardBinary, "/uninstalltunnelservice", cfg.Interface)
		if _, err := uninstallService.CombinedOutput(); err != nil {
			log.Infof("uninstalling tunnel service, this is ok: %v", err)
		} else {
			log.Infof("uninstalled tunnel service: %v")
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

func fileActuallyChanged(old, new []byte) bool {
	if old == nil || new == nil {
		return true
	}

	return !bytes.Equal(old, new)
}
