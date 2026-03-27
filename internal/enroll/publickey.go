package enroll

import (
	"encoding/base64"
	"fmt"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func NormalizeWireGuardPublicKey(raw string) (string, error) {
	canonicalCandidate := strings.TrimSpace(raw)
	if canonicalCandidate == "" {
		return "", fmt.Errorf("invalid wireguard public key: empty value")
	}

	canonicalKey, canonicalErr := wgtypes.ParseKey(canonicalCandidate)
	if canonicalErr == nil {
		return canonicalKey.String(), nil
	}

	decoded, decodeErr := base64.StdEncoding.DecodeString(canonicalCandidate)
	if decodeErr != nil {
		return "", fmt.Errorf("invalid wireguard public key: expected canonical key or base64-encoded canonical key")
	}

	legacyCandidate := strings.TrimSpace(string(decoded))
	legacyKey, legacyErr := wgtypes.ParseKey(legacyCandidate)
	if legacyErr != nil {
		return "", fmt.Errorf("invalid wireguard public key: expected canonical key or base64-encoded canonical key")
	}

	return legacyKey.String(), nil
}
