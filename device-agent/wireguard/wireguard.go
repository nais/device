package wireguard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/nais/device/device-agent/apiserver"
	"golang.org/x/crypto/curve25519"
)

func KeyToBase64(key []byte) []byte {
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(key)))
	base64.StdEncoding.Encode(dst, key)
	return dst
}

func Base64toKey(encoded []byte) ([]byte, error) {
	decoded := make([]byte, 32)
	_, err := base64.StdEncoding.Decode(decoded, encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding base64 key: %w", err)
	}

	return decoded, nil
}

func WgGenKey() []byte {
	var privateKey [32]byte

	n, err := rand.Read(privateKey[:])

	if err != nil || n != len(privateKey) {
		panic("Unable to generate random bytes")
	}

	privateKey[0] &= 248
	privateKey[31] = (privateKey[31] & 127) | 64
	return privateKey[:]
}

func WGPubKey(privateKeySlice []byte) []byte {
	var privateKey [32]byte
	var publicKey [32]byte
	copy(privateKey[:], privateKeySlice[:])

	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return publicKey[:]
}

func GenerateWireGuardPeers(gateways []apiserver.Gateway) string {
	peerTemplate := `[Peer]
PublicKey = %s
AllowedIPs = %s
Endpoint = %s
`
	var peers string

	for _, gateway := range gateways {
		allowedIPs := strings.Join(append(gateway.Routes, gateway.IP+"/32"), ",")
		peers += fmt.Sprintf(peerTemplate, gateway.PublicKey, allowedIPs, gateway.Endpoint)
	}

	return peers
}
