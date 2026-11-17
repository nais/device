package helper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nais/device/pkg/pb"
)

func TestApplyWireGuardConfig_InvalidPrivateKey(t *testing.T) {
	err := applyWireGuardConfig("does-not-exist", &pb.Configuration{PrivateKey: "not-a-valid-key"})
	require.ErrorContains(t, err, "parse private key")
}
