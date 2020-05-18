package device_agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nais/device/device-agent/config"
	"github.com/nais/device/device-agent/wireguard"
)

func (d *DeviceAgent) RunHelper(ctx context.Context) error {
	cmd := adminCommandContext(ctx, "./bin/device-agent-helper",
		"--interface", d.Config.Interface,
		"--tunnel-ip", d.Config.BootstrapConfig.TunnelIP,
		"--wireguard-binary", d.Config.WireGuardBinary,

		"--wireguard-go-binary", d.Config.WireGuardGoBinary,
		"--wireguard-config-path", d.Config.WireGuardConfigPath)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Start()
}

func GenerateBaseConfig(privateKey []byte, bootstrapConfig *config.BootstrapConfig) string {
	template := `[Interface]
PrivateKey = %s

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`
	return fmt.Sprintf(template, wireguard.KeyToBase64(privateKey), bootstrapConfig.PublicKey, bootstrapConfig.APIServerIP, bootstrapConfig.Endpoint)
}

func (d *DeviceAgent) setPlatformDefaults() {
	d.Config.WireGuardBinary = filepath.Join(d.Config.BinaryDir, "naisdevice-wg")
	d.Config.WireGuardGoBinary = filepath.Join(d.Config.BinaryDir, "naisdevice-wireguard-go")
}

func (d *DeviceAgent) platformPrerequisites() error {
	if err := ensureDirectories(d.Config.BinaryDir); err != nil {
		return fmt.Errorf("ensuring directory exists: %w", err)
	}

	if err := filesExist(d.Config.WireGuardGoBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}

	return nil
}
