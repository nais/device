package wireguard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/filesystem"
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

func GenerateWireGuardPeers(gateways map[string]*apiserver.Gateway) string {
	peerTemplate := `[Peer]
PublicKey = %s
AllowedIPs = %s
Endpoint = %s
`
	var peers string

	for _, gateway := range gateways {
		allowedIPs := gateway.IP+"/32"
		if gateway.IsHealthy() {
			allowedIPs += strings.Join(gateway.Routes, ",")

		}
		peers += fmt.Sprintf(peerTemplate, gateway.PublicKey, allowedIPs, gateway.Endpoint)
	}

	return peers
}

//TODO(jhrv): test
func EnsurePrivateKey(keyPath string) ([]byte, error) {
	if err := filesystem.FileMustExist(keyPath); os.IsNotExist(err) {
		if err := ioutil.WriteFile(keyPath, KeyToBase64(WgGenKey()), 0600); err != nil {
			return nil, fmt.Errorf("writing private key to disk: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("ensuring private key exists: %w", err)
	}

	privateKeyEncoded, err := ioutil.ReadFile(keyPath)
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
