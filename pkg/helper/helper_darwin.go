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
	wireGuardBinary   = "/Applications/naisdevice.app/Contents/MacOS/wg"
	wireGuardGoBinary = "/Applications/naisdevice.app/Contents/MacOS/wireguard-go"
)

type DarwinConfigurator struct {
	helperConfig Config
}

var _ OSConfigurator = &DarwinConfigurator{}

func New(helperConfig Config) *DarwinConfigurator {
	return &DarwinConfigurator{
		helperConfig: helperConfig,
	}
}

func (c *DarwinConfigurator) Prerequisites() error {
	if err := filesExist(wireGuardBinary, wireGuardGoBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}

	return nil
}

func (c *DarwinConfigurator) SyncConf(ctx context.Context, cfg *pb.Configuration) error {
	cmd := exec.CommandContext(ctx, wireGuardBinary, "syncconf", c.helperConfig.Interface, WireGuardConfigPath)
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("running syncconf: %w: %v", err, string(b))
	}

	return nil
}

func (c *DarwinConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) error {
	for _, gw := range gateways {
		for _, cidr := range gw.GetRoutes() {
			if strings.HasPrefix(cidr, TunnelNetworkPrefix) {
				// Don't add routes for the tunnel network, as the whole /21 net is already routed to utun
				continue
			}

			cmd := exec.CommandContext(ctx, "route", "-q", "-n", "add", "-inet", cidr, "-interface", c.helperConfig.Interface)
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

func (c *DarwinConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	if c.interfaceExists(ctx) {
		return nil
	}

	commands := [][]string{
		{wireGuardGoBinary, c.helperConfig.Interface},
		{"ifconfig", c.helperConfig.Interface, "inet", cfg.GetDeviceIP() + "/21", cfg.GetDeviceIP(), "add"},
		{"ifconfig", c.helperConfig.Interface, "mtu", "1360"},
		{"ifconfig", c.helperConfig.Interface, "up"},
		{"route", "-q", "-n", "add", "-inet", cfg.GetDeviceIP() + "/21", "-interface", c.helperConfig.Interface},
	}

	return runCommands(ctx, commands)
}

func (c *DarwinConfigurator) TeardownInterface(ctx context.Context) error {
	if !c.interfaceExists(ctx) {
		return nil
	}

	cmd := exec.CommandContext(ctx, "pkill", "-f", fmt.Sprintf("%s %s", wireGuardGoBinary, c.helperConfig.Interface))
	out, err := cmd.CombinedOutput()

	if err != nil {
		log.Errorf("teardown output: %v", string(out))
		return err
	}

	return nil
}

func (c *DarwinConfigurator) interfaceExists(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "pgrep", "-f", fmt.Sprintf("%s %s", wireGuardGoBinary, c.helperConfig.Interface))
	return cmd.Run() == nil
}
