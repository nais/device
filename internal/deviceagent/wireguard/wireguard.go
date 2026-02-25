package wireguard

import (
	"fmt"
	"os"

	"github.com/nais/device/internal/deviceagent/filesystem"
	"github.com/nais/device/internal/wireguard"
)

func EnsurePrivateKey(keyPath string) ([]byte, error) {
	if err := filesystem.FileMustExist(keyPath); os.IsNotExist(err) {
		key, err := wireguard.GenKey()
		if err != nil {
			return nil, fmt.Errorf("generating private key: %w", err)
		}
		if err := os.WriteFile(keyPath, wireguard.KeyToBase64(key), 0o600); err != nil {
			return nil, fmt.Errorf("writing private key to disk: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("ensuring private key exists: %w", err)
	}

	privateKeyEncoded, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key: %v", err)
	}

	privateKey, err := wireguard.Base64ToKey(privateKeyEncoded)
	if err != nil {
		return nil, fmt.Errorf("decoding private key: %v", err)
	}

	return privateKey, nil
}

func PublicKey(privateKey []byte) []byte {
	return wireguard.KeyToBase64(wireguard.PubKey(privateKey))
}
