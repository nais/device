package wireguard_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nais/device/internal/deviceagent/wireguard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestGenKey(t *testing.T) {
	key, err := wgtypes.GeneratePrivateKey()
	assert.NoError(t, err)
	assert.Len(t, key, 32)
	assert.Len(t, key.String(), 44)
}

func TestEnsurePrivateKey_CreatesNewKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "private.key")

	key, err := wireguard.EnsurePrivateKey(keyPath)
	require.NoError(t, err)
	assert.NotEqual(t, wgtypes.Key{}, key, "generated key should not be zero")

	// Verify the key was written to disk
	data, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, key.String(), string(data))
}

func TestEnsurePrivateKey_ReadsExistingKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "private.key")

	// Write a known key to disk
	original, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(keyPath, []byte(original.String()), 0o600))

	// EnsurePrivateKey should read it back
	key, err := wireguard.EnsurePrivateKey(keyPath)
	require.NoError(t, err)
	assert.Equal(t, original, key)
}

func TestEnsurePrivateKey_IdempotentAcrossCalls(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "private.key")

	key1, err := wireguard.EnsurePrivateKey(keyPath)
	require.NoError(t, err)

	key2, err := wireguard.EnsurePrivateKey(keyPath)
	require.NoError(t, err)

	assert.Equal(t, key1, key2, "same key should be returned across calls")
}
