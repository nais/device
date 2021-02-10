package device_helper

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nais/device/pkg/pb"

	log "github.com/sirupsen/logrus"
)

var (
	WireGuardGoBinary = filepath.Join("/", "Applications", "naisdevice.app", "Contents", "MacOS", "wireguard-go")
	WireGuardBinary   = filepath.Join("/", "Applications", "naisdevice.app", "Contents", "MacOS", "wg")
)

func Prerequisites() error {
	if err := filesExist(WireGuardBinary, WireGuardGoBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}
	return nil
}

func PlatformInit(cfg *Config) {
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

			cmd := exec.CommandContext(ctx, "route", "-q", "-n", "add", "-inet", cidr, "-interface", iface)
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Errorf("%v: %v", cmd, string(output))
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
		{WireGuardGoBinary, iface},
		{"ifconfig", iface, "inet", cfg.GetDeviceIP() + "/21", cfg.GetDeviceIP(), "add"},
		{"ifconfig", iface, "mtu", "1360"},
		{"ifconfig", iface, "up"},
		{"route", "-q", "-n", "add", "-inet", cfg.GetDeviceIP() + "/21", "-interface", iface},
	}

	return runCommands(ctx, commands)
}

func TeardownInterface(ctx context.Context, iface string) {
	if !interfaceExists(ctx, iface) {
		return
	}

	cmd := exec.CommandContext(ctx, "pkill", "-f", fmt.Sprintf("%s %s", WireGuardGoBinary, iface))
	out, err := cmd.CombinedOutput()

	if err != nil {
		log.Errorf("tearing down interface failed: %v: %v", cmd, err)
		log.Errorf("teardown output: %v", string(out))
	}

	return
}

func interfaceExists(ctx context.Context, iface string) bool {
	cmd := exec.CommandContext(ctx, "pgrep", "-f", fmt.Sprintf("%s %s", WireGuardGoBinary, iface))
	return cmd.Run() == nil
}
