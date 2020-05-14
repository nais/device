package wireguard

func GenerateBaseConfig(bootstrapConfig *BootstrapConfig, privateKey []byte) string {
	template := `[Interface]
PrivateKey = %s
MTU = 1360
Address = %s

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`
	return fmt.Sprintf(template, privateKey, bootstrapConfig.TunnelIP, bootstrapConfig.PublicKey, bootstrapConfig.APIServerIP, bootstrapConfig.Endpoint)
}

