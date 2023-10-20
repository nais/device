package wireguard_test

import (
	"testing"

	"github.com/nais/device/pkg/wireguard"
	"github.com/stretchr/testify/assert"
)

func TestParseDefaultInterfaceOutputV4(t *testing.T) {
	input := []byte(`1.1.1.1 via 13.37.96.1 dev ens160 src 13.37.96.69 uid 1001
    cache
`)

	ifName, ifIP, err := wireguard.ParseDefaultInterfaceOutputV4(input)

	assert.NoError(t, err)
	assert.Equal(t, "ens160", ifName)
	assert.Equal(t, "13.37.96.69", ifIP)
}

// TODO
func TestParseDefaultInterfaceOutputV6(t *testing.T) {
	t.Skip()
	input := []byte(`need real output to test`)

	ifName, ifIP, err := wireguard.ParseDefaultInterfaceOutputV6(input)

	assert.NoError(t, err)
	assert.Equal(t, "ens160", ifName)
	assert.Equal(t, "13::37", ifIP)
}
