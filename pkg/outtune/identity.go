package outtune

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"

	"github.com/nais/device/pkg/ioconvenience"
	"software.sslmate.com/src/go-pkcs12"
)

type identity struct {
	privateKey  *rsa.PrivateKey
	certificate *x509.Certificate
}

func (id *identity) SerializePEM(w io.Writer) error {
	ew := ioconvenience.NewErrorWriter(w)
	pem.Encode(ew, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(id.privateKey),
	})
	pem.Encode(ew, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: id.certificate.Raw,
	})

	_, err := ew.Status()
	return err
}

func (id *identity) SerializePKCS12(w io.Writer) error {
	pk12, err := pkcs12.Encode(rand.Reader, id.privateKey, id.certificate, nil, dummyPassword)
	if err != nil {
		return err
	}
	_, err = w.Write(pk12)
	return err
}
