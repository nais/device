package main_test

import (
	"testing"

	main "github.com/nais/device/cmd/device-agent-helper"
	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	wgConfig := `[Interface]
PrivateKey = aGJsW8OLpTS1/D74ZCd69V8wGnfiKzzMmzTgRUp/A2s=

[Peer]
PublicKey = FUwVtyvs8nIRx9RpUUEopkfV8idmHz9g9K/vf9MFOXI=
AllowedIPs = 10.255.240.1/32
Endpoint = 35.228.142.96:51820
[Peer]
PublicKey = 9SPK+RUUy3SQuHGmI8d9wQ2ennPfOZ7tbS3wNmKD0X4=
AllowedIPs = 10.255.240.7/32
Endpoint = 35.228.208.255:51820
[Peer]
PublicKey = 7jxld9JjACu15jTUYba8qXOmXGk31H9DfV3pN5nv5g4=
AllowedIPs = 20.190.128.0/18,40.126.0.0/18,10.255.240.14/32
Endpoint = 35.228.118.232:51820
`
	cidrs, err := main.ParseConfig(wgConfig)
	assert.NoError(t, err)
	expectedCIDRs := []string{"10.255.240.1/32", "10.255.240.7/32", "20.190.128.0/18", "40.126.0.0/18", "10.255.240.14/32"}
	assert.Equal(t, expectedCIDRs, cidrs)
}
