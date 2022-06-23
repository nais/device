package bootstrap

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/mail"
	"regexp"
)

const blockTypePublicKey = "PUBLIC KEY"

func (d *DeviceInfo) Parse() error {
	_, err := ParseOwner(d.Owner)
	if err != nil {
		return err
	}

	err = ParseSerial(d.Serial)
	if err != nil {
		return err
	}

	err = parsePublicKey(d.PublicKey)
	if err != nil {
		return fmt.Errorf("public key: %v", err)
	}

	return nil
}

func ParseOwner(owner string) (string, error) {
	as, err := mail.ParseAddress(owner)
	if err != nil {
		return "", err
	}
	return as.Address, nil
}

func ParseSerial(serial string) error {
	if serial == "" {
		return fmt.Errorf("empty serial not allowed")
	}

	re := regexp.MustCompile("^[a-zA-Z\\d]*$")
	if !re.MatchString(serial) {
		return fmt.Errorf("serial: %v", serial)
	}
	return nil
}

func parsePublicKey(pemString string) error {
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return fmt.Errorf("no key found")
	}

	switch block.Type {
	case blockTypePublicKey:
		_, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported key type %q", block.Type)
	}
	return nil
}
