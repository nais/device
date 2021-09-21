package wireguard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"golang.org/x/crypto/curve25519"

	"github.com/nais/device/pkg/device-agent/filesystem"
	"github.com/nais/device/pkg/pb"
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

// TODO(jhrv): test
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

var wireGuardTemplateGateway = `[Peer]
PublicKey = %s
AllowedIPs = %s
Endpoint = %s

`

func MarshalGateway(w io.Writer, x *pb.Gateway) (int, error) {
	routes := append(x.GetRoutes(), x.GetIp())
	return fmt.Fprintf(w, wireGuardTemplateGateway, x.GetPublicKey(), strings.Join(routes, ","), x.GetEndpoint())
}

func Marshal(w io.Writer, x *pb.Configuration) (int, error) {
	gateways := x.GetGateways()[:]
	if gateways != nil {
		// Sort gateways here to let windows helper detect changes in, and prevent unnecessary restarts
		sort.Slice(gateways, func(i, j int) bool {
			return strings.Compare(gateways[i].Name, gateways[j].Name) < 0
		})
	}

	mw := multiWriter{w: w}

	_, _ = MarshalHeader(w, x)
	for _, gw := range gateways {
		_, _ = MarshalGateway(mw, gw)
	}

	return mw.Status()
}
