package wireguard

import (
	"fmt"
	"os"

	"github.com/nais/device/internal/deviceagent/filesystem"
	"github.com/nais/device/internal/wireguard"
)

func KeyToBase64(key []byte) []byte {
	return wireguard.KeyToBase64(key)
}

func Base64toKey(encoded []byte) ([]byte, error) {
	return wireguard.Base64ToKey(encoded)
}

func WgGenKey() []byte {
	key, err := wireguard.GenKey()
	if err != nil {
		panic("Unable to generate random bytes")
	}
	return []byte(key)
}

func WGPubKey(privateKeySlice []byte) []byte {
	return wireguard.PubKey(privateKeySlice)
}

func EnsurePrivateKey(keyPath string) ([]byte, error) {
	if err := filesystem.FileMustExist(keyPath); os.IsNotExist(err) {
		if err := os.WriteFile(keyPath, KeyToBase64(WgGenKey()), 0o600); err != nil {
			return nil, fmt.Errorf("writing private key to disk: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("ensuring private key exists: %w", err)
	}

	privateKeyEncoded, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key: %v", err)
	}

	privateKey, err := Base64toKey(privateKeyEncoded)
	if err != nil {
		return nil, fmt.Errorf("decoding private key: %v", err)
	}

	return privateKey, nil
}

func PublicKey(privateKey []byte) []byte {
	return KeyToBase64(WGPubKey(privateKey))
}
