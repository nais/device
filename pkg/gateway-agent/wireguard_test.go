package gateway_agent_test

import (
	"bytes"
	"testing"

	gateway_agent "github.com/nais/device/pkg/gateway-agent"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
)

func TestWriteWireGuardBase(t *testing.T) {
	cfg := gateway_agent.Config{
		PrivateKey:          "abc",
		DeviceIP:            "127.0.0.1",
		APIServerEndpoint:   "endpoint",
		APIServerPrivateIP:  "10.255.240.1",
		APIServerPublicKey:  "pubkey",
		PrometheusPublicKey: "prom",
		PrometheusTunnelIP:  "10.255.247.254",
	}

	buf := new(bytes.Buffer)
	err := cfg.WriteWireGuardBase(buf)

	assert.NoError(t, err)

	expected :=
		`[Interface]
PrivateKey = abc
ListenPort = 51820

[Peer] # apiserver
PublicKey = pubkey
AllowedIPs = 10.255.240.1/32
Endpoint = endpoint

[Peer] # prometheus
PublicKey = prom
AllowedIPs = 10.255.247.254/32

`
	assert.Equal(t, expected, buf.String())

}

func TestWriteWireGuardPeers(t *testing.T) {
	devices := []*pb.Device{
		{
			PublicKey: "pub1",
			Ip:        "ip1",
		},
		{
			PublicKey: "pub2",
			Ip:        "ip2",
		},
	}

	buf := new(bytes.Buffer)
	err := gateway_agent.WriteWireGuardPeers(buf, pb.DevicesAsPeers(devices))
	assert.NoError(t, err)

	expected := `[Peer]
PublicKey = pub1
AllowedIPs = ip1

[Peer]
PublicKey = pub2
AllowedIPs = ip2

`
	assert.Equal(t, expected, buf.String())

}
