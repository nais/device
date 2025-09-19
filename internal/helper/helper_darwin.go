package helper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nais/device/pkg/pb"
)

type DarwinConfigurator struct {
	helperConfig      Config
	wireGuardBinary   string
	wireGuardGoBinary string
}

var _ OSConfigurator = &DarwinConfigurator{}

func New(helperConfig Config) *DarwinConfigurator {
	return &DarwinConfigurator{
		helperConfig: helperConfig,
	}
}

func pathWithFallBacks(binary string, possiblePaths ...string) (string, error) {
	if p, err := exec.LookPath(binary); err == nil {
		return p, nil
	}

	for _, p := range possiblePaths {
		if s, err := os.Stat(p); err == nil && !s.IsDir() {
			return p, nil
		}
	}

	return "", fmt.Errorf("%q not found in PATH or any of %+v", binary, possiblePaths)
}

func (c *DarwinConfigurator) Prerequisites() error {
	var err error

	c.wireGuardBinary, err = pathWithFallBacks("wg", "/usr/local/bin/wg", "/opt/homebrew/bin/wg", "/usr/bin/wg")
	if err != nil {
		return fmt.Errorf("look for wg: %w", err)
	}

	c.wireGuardGoBinary, err = pathWithFallBacks("wireguard-go", "/usr/local/bin/wireguard-go", "/opt/homebrew/bin/wireguard-go", "/usr/bin/wireguard-go")
	if err != nil {
		return fmt.Errorf("look for wireguard-go: %w", err)
	}

	return nil
}

func (c *DarwinConfigurator) SyncConf(ctx context.Context, cfg *pb.Configuration) error {
	cmd := exec.CommandContext(ctx, c.wireGuardBinary, "syncconf", c.helperConfig.Interface, c.helperConfig.WireGuardConfigPath)
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("running syncconf: %w: %v", err, string(b))
	}

	return nil
}

func (c *DarwinConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) (int, error) {
	routesAdded := 0
	for _, gw := range gateways {
		applyRoute := func(cidr, family string) error {
			cmd := exec.CommandContext(ctx, "route", "-q", "-n", "add", family, cidr, "-interface", c.helperConfig.Interface)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("executing %v, err: %w, stderr: %s", cmd, err, string(output))
			}

			return nil
		}

		for _, cidr := range gw.GetRoutesIPv4() {
			if strings.HasPrefix(cidr, TunnelNetworkPrefix) {
				// Don't add routes for the tunnel network, as the whole /21 net is already routed to utun
				continue
			}
			err := applyRoute(cidr, "-inet")
			if err != nil {
				return routesAdded, err
			}
			routesAdded++
		}

		for _, cidr := range gw.GetRoutesIPv6() {
			err := applyRoute(cidr, "-inet6")
			if err != nil {
				return routesAdded, err
			}
			routesAdded++
		}
	}

	return routesAdded, nil
}

func (c *DarwinConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	if c.interfaceExists(ctx) {
		return nil
	}

	commands := [][]string{
		{c.wireGuardGoBinary, c.helperConfig.Interface},
		{"ifconfig", c.helperConfig.Interface, "inet", cfg.GetDeviceIPv4() + "/21", cfg.GetDeviceIPv4(), "alias"},
		{"ifconfig", c.helperConfig.Interface, "inet6", cfg.GetDeviceIPv6() + "/64", "alias"},
		{"ifconfig", c.helperConfig.Interface, "mtu", "1360"},
		{"ifconfig", c.helperConfig.Interface, "up"},
		{"route", "-q", "-n", "add", "-inet", cfg.GetDeviceIPv4() + "/21", "-interface", c.helperConfig.Interface},
	}

	return runCommands(ctx, commands)
}

func (c *DarwinConfigurator) TeardownInterface(ctx context.Context) error {
	if !c.interfaceExists(ctx) {
		return nil
	}

	cmd := exec.CommandContext(ctx, "pkill", "-f", fmt.Sprintf("%s %s", c.wireGuardGoBinary, c.helperConfig.Interface))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("teardown failed: %w, stderr: %s", err, string(out))
	}

	return nil
}

func (c *DarwinConfigurator) interfaceExists(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "pgrep", "-f", fmt.Sprintf("%s %s", c.wireGuardGoBinary, c.helperConfig.Interface))
	return cmd.Run() == nil
}
