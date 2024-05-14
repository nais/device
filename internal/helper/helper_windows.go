package helper

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/nais/device/internal/pb"
)

const wireGuardBinary = `c:\Program Files\WireGuard\wireguard.exe`

type WindowsConfigurator struct {
	helperConfig       Config
	oldWireGuardConfig []byte
	wgNeedsRestart     bool
}

var _ OSConfigurator = &WindowsConfigurator{}

func New(helperConfig Config) *WindowsConfigurator {
	return &WindowsConfigurator{
		helperConfig: helperConfig,
	}
}

func (configurator *WindowsConfigurator) Prerequisites() error {
	if err := filesExist(wireGuardBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}

	return nil
}

func interfaceExists(ctx context.Context, iface string) bool {
	queryService := exec.CommandContext(ctx, "sc", "query", tunnelName(iface))
	if err := queryService.Run(); err != nil {
		return false
	} else {
		return true
	}
}

func (configurator *WindowsConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	if interfaceExists(ctx, configurator.helperConfig.Interface) {
		return nil
	}

	installService := exec.CommandContext(ctx, wireGuardBinary, "/installtunnelservice", configurator.helperConfig.WireGuardConfigPath)
	if b, err := installService.CombinedOutput(); err != nil {
		return fmt.Errorf("installing tunnel service: %v: %v", err, string(b))
	} else {
		// log.Infof("installed tunnel service, sleeping 6 sec to let it finish")
		time.Sleep(6 * time.Second)
	}

	configurator.wgNeedsRestart = false

	return nil
}

func (configurator *WindowsConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) (int, error) {
	return 0, nil
}

func (configurator *WindowsConfigurator) SyncConf(ctx context.Context, cfg *pb.Configuration) error {
	newWireGuardConfig, err := os.ReadFile(configurator.helperConfig.WireGuardConfigPath)
	if err != nil {
		return fmt.Errorf("reading WireGuard config file: %w", err)
	}

	defer func() {
		configurator.oldWireGuardConfig = newWireGuardConfig
		configurator.wgNeedsRestart = true
	}()

	if !configurator.wgNeedsRestart {
		return nil
	}

	if fileActuallyChanged(configurator.oldWireGuardConfig, newWireGuardConfig) {
		// log.Debugf("old: %s", string(configurator.oldWireGuardConfig))
		// log.Debugf("new: %s", string(newWireGuardConfig))

		commands := [][]string{
			{"net", "stop", tunnelName(configurator.helperConfig.Interface)},
			{"net", "start", tunnelName(configurator.helperConfig.Interface)},
		}

		return runCommands(ctx, commands)
	}

	return nil
}

func (configurator *WindowsConfigurator) TeardownInterface(ctx context.Context) error {
	if !interfaceExists(ctx, configurator.helperConfig.Interface) {
		// log.Info("no interface")
		return nil
	}

	uninstallService := exec.CommandContext(ctx, wireGuardBinary, "/uninstalltunnelservice", configurator.helperConfig.Interface)

	b, err := uninstallService.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uninstalling tunnel service: %v: %v", err, string(b))
	} else {
		// log.Infof("uninstalled tunnel service (sleeping 3 sec to let it finish)")
		time.Sleep(3 * time.Second)
	}

	return nil
}

func tunnelName(interfaceName string) string {
	return fmt.Sprintf("WireGuardTunnel$%s", interfaceName)
}

func fileActuallyChanged(old, new []byte) bool {
	if old == nil || new == nil {
		return true
	}

	return !bytes.Equal(old, new)
}
