package wireguard

import (
	"fmt"

	"github.com/nais/device/device-agent/bootstrapper"
)

func GenerateBaseConfig(bootstrapConfig *bootstrapper.BootstrapConfig, privateKey []byte) string {
	template := `[Interface]
PrivateKey = %s
MTU = 1360
Address = %s

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`
	return fmt.Sprintf(template, privateKey, bootstrapConfig.DeviceIP, bootstrapConfig.PublicKey, bootstrapConfig.APIServerIP, bootstrapConfig.Endpoint)
}
