package device_agent

import (
	"context"
	"fmt"
	"os"

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
