package main_test

import (
	main "github.com/nais/device/cmd/gateway-agent"

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseDefaultInterfaceOutput(t *testing.T) {
	input := []byte(`1.1.1.1 via 13.37.96.1 dev ens160 src 13.37.96.69 uid 1001
    cache
`)

	ifName, ifIP, err := main.ParseDefaultInterfaceOutput(input)

	assert.NoError(t, err)
	assert.Equal(t, "ens160", ifName)
	assert.Equal(t, "13.37.96.69", ifIP)
}
