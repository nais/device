package gateway_agent_test

import (
	"testing"

	gateway_agent "github.com/nais/device/pkg/gateway-agent"
	"github.com/stretchr/testify/assert"
)

func TestParseDefaultInterfaceOutput(t *testing.T) {
	input := []byte(`1.1.1.1 via 13.37.96.1 dev ens160 src 13.37.96.69 uid 1001
    cache
`)

	ifName, ifIP, err := gateway_agent.ParseDefaultInterfaceOutput(input)

	assert.NoError(t, err)
	assert.Equal(t, "ens160", ifName)
	assert.Equal(t, "13.37.96.69", ifIP)
}
