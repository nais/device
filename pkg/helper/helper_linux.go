package helper

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nais/device/pkg/pb"
)

const (
	wireguardBinary = "/usr/bin/wg"
)

func New(helperConfig Config) *LinuxConfigurator {
	return &LinuxConfigurator{
		helperConfig: helperConfig,
	}
}

type LinuxConfigurator struct {
	helperConfig Config
}

var _ OSConfigurator = &LinuxConfigurator{}

func (c *LinuxConfigurator) Prerequisites() error {
	if err := filesExist(wireguardBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}

	return nil
}

func (c *LinuxConfigurator) SyncConf(ctx context.Context, cfg *pb.Configuration) error {
	cmd := exec.CommandContext(ctx, wireguardBinary, "syncconf", c.helperConfig.Interface, c.helperConfig.WireGuardConfigPath)
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("running syncconf: %w: %v", err, string(b))
	}

	return nil
}

func (c *LinuxConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) error {
	for _, gw := range gateways {
		// For Linux we can handle ipv4/6 addreses the same - the `ip` utility handles this for us
		for _, cidr := range append(gw.GetRoutesIPv4(), gw.GetRoutesIPv6()...) {
			if strings.HasPrefix(cidr, TunnelNetworkPrefix) {
				// Don't add routes for the tunnel network, as the whole /21 net is already routed to utun
				continue
			}

			cidr = strings.TrimSpace(cidr)

			cmd := exec.CommandContext(ctx, "ip", "route", "add", cidr, "dev", c.helperConfig.Interface)
			output, err := cmd.CombinedOutput()
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == 2 && strings.Contains(string(output), "File exists") {
					continue
				}
				return fmt.Errorf("executing %v: %w, stderr: %s", cmd, exitErr, string(output))
			}
		}
	}

	return nil
}

func (c *LinuxConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	if c.interfaceExists(ctx) {
		return nil
	}

	commands := [][]string{
		{"ip", "link", "add", "dev", c.helperConfig.Interface, "type", "wireguard"},
		{"ip", "link", "set", "mtu", "1360", "up", "dev", c.helperConfig.Interface},
		{"ip", "address", "add", "dev", c.helperConfig.Interface, cfg.DeviceIPv4 + "/21"},
		{"ip", "address", "add", "dev", c.helperConfig.Interface, cfg.DeviceIPv6 + "/64"},
	}

	return runCommands(ctx, commands)
}

func (c *LinuxConfigurator) TeardownInterface(ctx context.Context) error {
	if !c.interfaceExists(ctx) {
		return nil
	}

	cmd := exec.CommandContext(ctx, "ip", "link", "del", c.helperConfig.Interface)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("teardown failed: %w, stderr: %s", err, string(out))
	}

	return nil
}

func (c *LinuxConfigurator) interfaceExists(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "ip", "link", "show", "dev", c.helperConfig.Interface)
	return cmd.Run() == nil
}
