package device_agent

import (
	"context"
	"fmt"
	"os"
)

func (d *DeviceAgent) runHelper(ctx context.Context) error {
	cmd := adminCommandContext(ctx, "./bin/device-agent-helper.exe",
		"--interface", d.Config.Interface,
		"--wireguard-binary", d.Config.WireGuardBinary,
		"--wireguard-config-path", d.Config.WireGuardConfigPath)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Start()
}

func (d *DeviceAgent) GenerateBaseConfig() string {
	template := `[Interface]
PrivateKey = %s
MTU = 1360
Address = %s

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`
	return fmt.Sprintf(template, d.PrivateKey, d.BootstrapConfig.TunnelIP, d.BootstrapConfig.PublicKey, d.BootstrapConfig.APIServerIP, d.BootstrapConfig.Endpoint)
}
