package wireguard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

func WireGuardGenerateKeyPair() (string, string) {
	var publicKeyArray [32]byte
	var privateKeyArray [32]byte

	n, err := rand.Read(privateKeyArray[:])

	if err != nil || n != len(privateKeyArray) {
		panic("Unable to generate random bytes")
	}

	privateKeyArray[0] &= 248
	privateKeyArray[31] = (privateKeyArray[31] & 127) | 64

	curve25519.ScalarBaseMult(&publicKeyArray, &privateKeyArray)

	publicKeyString := base64.StdEncoding.EncodeToString(publicKeyArray[:])
	privateKeyString := base64.StdEncoding.EncodeToString(privateKeyArray[:])

	return publicKeyString, privateKeyString
}

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
	copy(privateKeySlice[:], privateKey[:])

	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return publicKey[:]
}
