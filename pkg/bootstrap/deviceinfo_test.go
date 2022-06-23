package bootstrap

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeviceInfo(t *testing.T) {

	for _, test := range []struct {
		name         string
		device       DeviceInfo
		error        bool
		errorMessage string
	}{
		{
			name: "sunnyside - everything's good",
			device: DeviceInfo{
				Serial:    "K33HP66KLO8P",
				PublicKey: pubKey,
				Platform:  "macOS",
				Owner:     "john.Doe@naisdevice.io",
			},
		},
		{
			name:         "malformed email",
			error:        true,
			errorMessage: "mail: missing '@' or angle-addr",
			device: DeviceInfo{
				Serial:    "K33HP66KLO8P",
				PublicKey: pubKey,
				Platform:  "macOS",
				Owner:     "john.Doe-naisdevice.io",
			},
		},
		{
			name:         "malformed serial",
			error:        true,
			errorMessage: "serial: K33HP66KLO#P",
			device: DeviceInfo{
				Serial:    "K33HP66KLO#P",
				PublicKey: pubKey,
				Platform:  "macOS",
				Owner:     "john.Doe@naisdevice.io",
			},
		},
		{
			name:         "malformed public key",
			error:        true,
			errorMessage: "public key: no key found",
			device: DeviceInfo{
				Serial:    "K33HP66KLO6P",
				PublicKey: "some malformed key",
				Platform:  "macOS",
				Owner:     "john.Doe@naisdevice.io",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.device.Parse()
			if !test.error {
				assert.NoError(t, err)
			} else {
				fmt.Println(err)
				assert.ErrorContains(t, err, test.errorMessage)
			}
		})
	}
}

const (
	pubKey = `
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEPv95/7odwATyyKVElyI8Os7jvC8K
Ab/dPGpiMDXw0kLOH4AnLnlBotW85O7Huqlqf9SRcCFFIaTbJbLAz8O5eg==
-----END PUBLIC KEY-----
`
)
