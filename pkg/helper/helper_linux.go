package helper

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nais/device/pkg/pb"

	log "github.com/sirupsen/logrus"
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
	cmd := exec.CommandContext(ctx, wireguardBinary, "syncconf", c.helperConfig.Interface, WireGuardConfigPath)
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("running syncconf: %w: %v", err, string(b))
	}

	return nil
}

func (c *LinuxConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) error {
	for _, gw := range gateways {
		for _, cidr := range gw.GetRoutes() {
			if strings.HasPrefix(cidr, TunnelNetworkPrefix) {
				// Don't add routes for the tunnel network, as the whole /21 net is already routed to utun
				continue
			}

			cmd := exec.CommandContext(ctx, "ip", "-4", "route", "add", cidr, "dev", c.helperConfig.Interface)
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

func (c *LinuxConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	if c.interfaceExists(ctx) {
		return nil
	}

	commands := [][]string{
		{"ip", "link", "add", "dev", c.helperConfig.Interface, "type", "wireguard"},
		{"ip", "link", "set", "mtu", "1360", "up", "dev", c.helperConfig.Interface},
		{"ip", "address", "add", "dev", c.helperConfig.Interface, cfg.DeviceIP + "/21"},
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
		log.Errorf("teardown output: %v", string(out))
		return err
	}

	return nil
}

func (c *LinuxConfigurator) interfaceExists(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "ip", "link", "show", "dev", c.helperConfig.Interface)
	return cmd.Run() == nil
}
