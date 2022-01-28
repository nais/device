package outtune

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
)

type identity struct {
	privateKey  *rsa.PrivateKey
	certificate string
}

func (id *identity) SerializePEM(w io.Writer) error {
	ew := &errorWriter{w: w}
	pem.Encode(ew, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(id.privateKey),
	})
	ew.Write([]byte(id.certificate))
	return ew.Error()
}
