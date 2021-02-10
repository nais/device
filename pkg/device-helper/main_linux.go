package device_helper

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nais/device/pkg/pb"

	log "github.com/sirupsen/logrus"
)

const (
	WireGuardBinary = "/usr/bin/wg"
)

func Prerequisites() error {
	if err := filesExist(WireGuardBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}

	return nil
}

func syncConf(ctx context.Context, cfg Config) error {
	cmd := exec.CommandContext(ctx, WireGuardBinary, "syncconf", cfg.Interface, cfg.WireGuardConfigPath)
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("running syncconf: %w: %v", err, string(b))
	}

	return nil
}

func setupRoutes(ctx context.Context, gateways []*pb.Gateway, iface string) error {
	for _, gw := range gateways {
		for _, cidr := range gw.GetRoutes() {
			if strings.HasPrefix(cidr, TunnelNetworkPrefix) {
				// Don't add routes for the tunnel network, as the whole /21 net is already routed to utun
				continue
			}

			cmd := exec.CommandContext(ctx, "ip", "-4", "route", "add", cidr, "dev", iface)
			output, err := cmd.CombinedOutput()
			if exitErr, ok := err.(*exec.ExitError); ok {
				log.Debugf("Command: %v, exit code: %v, output: %v", cmd, exitErr.ExitCode(), string(output))
				if exitErr.ExitCode() == 2 && strings.Contains(string(output), "File exists") {
					log.Debug("Assuming route already exists")
					continue
				}
				return fmt.Errorf("executing %v: %w", cmd, err)
			}
			log.Debugf("%v: %v", cmd, string(output))
		}
	}

	return nil
}

func setupInterface(ctx context.Context, iface string, cfg *pb.Configuration) error {
	if interfaceExists(ctx, iface) {
		return nil
	}

	commands := [][]string{
		{"ip", "link", "add", "dev", iface, "type", "wireguard"},
		{"ip", "link", "set", "mtu", "1360", "up", "dev", iface},
		{"ip", "address", "add", "dev", iface, cfg.DeviceIP + "/21"},
	}

	return runCommands(ctx, commands)
}

func TeardownInterface(ctx context.Context, iface string) {
	if !interfaceExists(ctx, iface) {
		return
	}

	cmd := exec.CommandContext(ctx, "ip", "link", "del", iface)
	out, err := cmd.CombinedOutput()

	if err != nil {
		log.Errorf("tearing down interface failed: %v: %v", cmd, err)
		log.Errorf("teardown output: %v", string(out))
	}
}

func interfaceExists(ctx context.Context, iface string) bool {
	cmd := exec.CommandContext(ctx, "ip", "link", "show", "dev", iface)
	return cmd.Run() == nil
}
