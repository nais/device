package main

import (
	"context"
	"fmt"
	"github.com/nais/device/pkg/bootstrap"
	"github.com/nais/device/pkg/logger"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	WireGuardBinary   = "/usr/bin/wg"
	ProductSerialPath = "/sys/devices/virtual/dmi/id/product_serial"
)

func prerequisites() error {
	if err := filesExist(WireGuardBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}
	return nil
}

func platformFlags(cfg *Config) {}
func platformInit(cfg *Config) {
	logger.SetupDeviceLogger(cfg.LogLevel, filepath.Join("/", "var", "log", "device-agent-helper.log"))
	if err := extractProductSerial(ProductSerialPath); err != nil {
		log.Error(err)
	}

}

func syncConf(cfg Config, ctx context.Context) error {
	cmd := exec.CommandContext(ctx, WireGuardBinary, "syncconf", cfg.Interface, cfg.WireGuardConfigPath)
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("running syncconf: %w: %v", err, string(b))
	}

	configFileBytes, err := ioutil.ReadFile(cfg.WireGuardConfigPath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	cidrs, err := ParseConfig(string(configFileBytes))
	if err != nil {
		return fmt.Errorf("parsing WireGuard config: %w", err)
	}

	if err := setupRoutes(ctx, cidrs, cfg.Interface); err != nil {
		return fmt.Errorf("setting up routes: %w", err)
	}

	return nil
}

func setupRoutes(ctx context.Context, cidrs []string, interfaceName string) error {
	for _, cidr := range cidrs {
		cmd := exec.CommandContext(ctx, "ip", "-4", "route", "add", cidr, "dev", interfaceName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				log.Debugf("Command: %v, exit code: %v, output: %v", cmd, exitErr.ExitCode(), string(output))
				if exitErr.ExitCode() == 2 && strings.Contains(string(output), "File exists") {
					log.Debug("Assuming route already exists")
					continue
				}
			}

			return fmt.Errorf("executing %v: %w", cmd, err)
		}
		log.Debugf("%v: %v", cmd, string(output))
	}
	return nil
}

func setupInterface(ctx context.Context, cfg Config, bootstrapConfig *bootstrap.Config) error {
	if err := exec.Command("ip", "link", "del", cfg.Interface).Run(); err != nil {
		log.Infof("pre-deleting WireGuard interface (ok if this fails): %v", err)
	}

	commands := [][]string{
		{"ip", "link", "add", "dev", cfg.Interface, "type", "wireguard"},
		{"ip", "link", "set", "mtu", "1360", "up", "dev", cfg.Interface},
		{"ip", "address", "add", "dev", cfg.Interface, bootstrapConfig.DeviceIP + "/21"},
	}

	return runCommands(ctx, commands)
}

func teardownInterface(ctx context.Context, cfg Config) {
	cmd := exec.CommandContext(ctx, "ip", "link", "del", cfg.Interface)
	if err := cmd.Run(); err != nil {
		log.Errorf("Tearing down interface: %v", err)
	}
}

func extractProductSerial(cfgpath string) error {
	target := filepath.Join(cfg.ConfigPath, "product_serial")
	serialBytes, err := ioutil.ReadFile(ProductSerialPath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	if err := ioutil.WriteFile(target, serialBytes, 0644); err != nil {
		return fmt.Errorf("Writing product serial to disk: %v", err)
	} else {
		log.Debugf("Successfully wrote product serial to: %v", target)
	}
	return nil
}

func uninstallService()         {}
func installService(cfg Config) {}
