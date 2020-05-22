package wireguard

import (
	"fmt"

	"github.com/nais/device/pkg/bootstrap"
)

func GenerateBaseConfig(bootstrapConfig *bootstrap.Config, privateKey []byte) string {
	template := `[Interface]
PrivateKey = %s
MTU = 1360
Address = %s

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`
	return fmt.Sprintf(template, KeyToBase64(privateKey), bootstrapConfig.DeviceIP, bootstrapConfig.PublicKey, bootstrapConfig.APIServerIP, bootstrapConfig.TunnelEndpoint)
}
