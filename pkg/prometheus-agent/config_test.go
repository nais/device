package prometheusagent_test

import (
	"bytes"
	"testing"

	prometheusagent "github.com/nais/device/pkg/prometheus-agent"
	"github.com/stretchr/testify/assert"
)

func TestWriteWireGuardBase(t *testing.T) {
	cfg := prometheusagent.Config{
		PrivateKey:         "abc",
		TunnelIP:           "127.0.0.1",
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
