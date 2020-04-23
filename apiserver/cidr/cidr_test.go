package cidr_test

import (
	"testing"

	"github.com/nais/device/apiserver/cidr"
	"github.com/stretchr/testify/assert"
)

func TestFindAvailableIP(t *testing.T) {
	t.Run("finds the lowest ip address in range", func(t *testing.T) {
		availableIP, err := cidr.FindAvailableIP("10.0.0.0/30", []string{"10.0.0.2"})
		assert.NoError(t, err)
		assert.Equal(t, "10.0.0.1", availableIP)
	})

	t.Run("fills gaps", func(t *testing.T) {
		availableIP, err := cidr.FindAvailableIP("10.0.0.0/30", []string{"10.0.0.1", "10.0.0.3"})
		assert.NoError(t, err)
		assert.Equal(t, "10.0.0.2", availableIP)
	})

	t.Run("returns an error if noone is available", func(t *testing.T) {
		_, err := cidr.FindAvailableIP("10.0.0.1/30", []string{"10.0.0.1", "10.0.0.2"})
		assert.Error(t, err)
	})
}
