package gatewayconfigurer_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nais/device/pkg/apiserver/gatewayconfigurer"
)

func TestToCIDRStringSlice(t *testing.T) {
	cidr := "1.2.3.4"
	cidrStringSlice := gatewayconfigurer.ToCIDRStringSlice([]gatewayconfigurer.Route{{CIDR: cidr}})
	assert.Len(t, cidrStringSlice, 1)
	assert.Equal(t, cidr, cidrStringSlice[0])
}
