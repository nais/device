package config_test

import (
	"bytes"
	"testing"

	"github.com/nais/device/internal/prometheus-agent/config"
	"github.com/stretchr/testify/assert"
)

func TestWriteWireGuardBase(t *testing.T) {
	cfg := config.Config{
		PrivateKey:         "abc",
		DeviceIPv4:         "127.0.0.1",
		APIServerEndpoint:  "endpoint",
		APIServerTunnelIP:  "10.255.240.1",
		APIServerPublicKey: "pubkey",
	}

	buf := new(bytes.Buffer)
	err := cfg.WriteWireGuardBase(buf)

	assert.NoError(t, err)

	expected := `[Interface]
PrivateKey = abc
ListenPort = 51820

[Peer] # apiserver
PublicKey = pubkey
AllowedIPs = 10.255.240.1/32
Endpoint = endpoint

`
	assert.Equal(t, expected, buf.String())
}
