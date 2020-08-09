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

var (
	WireGuardGoBinary = filepath.Join("/", "Applications", "naisdevice.app", "Contents", "MacOS", "wireguard-go")
	WireGuardBinary   = filepath.Join("/", "Applications", "naisdevice.app", "Contents", "MacOS", "wg")
)

func prerequisites() error {
	if err := filesExist(WireGuardBinary, WireGuardGoBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}
	return nil
}

func platformFlags(cfg *Config) {}
func platformInit(cfg *Config) {
	logger.SetupDeviceLogger(cfg.LogLevel, filepath.Join("/", "Library", "Logs", "device-agent-helper.log"))
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
		if strings.HasPrefix(cidr, TunnelNetworkPrefix) {
			// Don't add routes for the tunnel network, as the whole /21 net is already routed to utun
			continue
		}

		cmd := exec.CommandContext(ctx, "route", "-q", "-n", "add", "-inet", cidr, "-interface", interfaceName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Errorf("%v: %v", cmd, string(output))
			return fmt.Errorf("executing %v: %w", cmd, err)
		}
		log.Debugf("%v: %v", cmd, string(output))
	}
	return nil
}

func setupInterface(ctx context.Context, cfg Config, bootstrapConfig *bootstrap.Config) error {
	if interfaceExists(ctx, cfg) {
		return nil
	}

	ip := bootstrapConfig.DeviceIP
	commands := [][]string{
		{WireGuardGoBinary, cfg.Interface},
		{"ifconfig", cfg.Interface, "inet", ip + "/21", ip, "add"},
		{"ifconfig", cfg.Interface, "mtu", "1360"},
		{"ifconfig", cfg.Interface, "up"},
		{"route", "-q", "-n", "add", "-inet", ip + "/21", "-interface", cfg.Interface},
	}

	return runCommands(ctx, commands)
}

func teardownInterface(ctx context.Context, cfg Config) {
	if !interfaceExists(ctx, cfg) {
		return
	}

	cmd := exec.CommandContext(ctx, "pkill", "-f", fmt.Sprintf("%s %s", WireGuardGoBinary, cfg.Interface))
	out, err := cmd.CombinedOutput()

	if err != nil {
		log.Infof("tearing down interface failed: %v: %v", cmd, err)
		log.Infof("teardown output: %v", string(out))
	}

	return
}

func interfaceExists(ctx context.Context, cfg Config) bool {
	cmd := exec.CommandContext(ctx, "pgrep", "-f", fmt.Sprintf("%s %s", WireGuardGoBinary, cfg.Interface))
	return cmd.Run() == nil
}

func uninstallService()         {}
func installService(cfg Config) {}
