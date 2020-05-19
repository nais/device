// +build linux darwin

package wireguard

import (
	"fmt"
	"github.com/nais/device/pkg/bootstrap"
)

func GenerateBaseConfig(bootstrapConfig *bootstrap.Config, privateKey []byte) string {
	template := `[Interface]
PrivateKey = %s

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`
	return fmt.Sprintf(template, KeyToBase64(privateKey), bootstrapConfig.PublicKey, bootstrapConfig.APIServerIP, bootstrapConfig.TunnelEndpoint)
}
