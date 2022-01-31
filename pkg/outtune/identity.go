package outtune

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"

	pkcs12 "software.sslmate.com/src/go-pkcs12"
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

func (id *identity) SerializePKCS12(w io.Writer) error {
	block, _ := pem.Decode([]byte(id.certificate))
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	pk12, err := pkcs12.Encode(rand.Reader, id.privateKey, cert, nil, dummyPassword)
	if err != nil {
		return err
	}
	_, err = w.Write(pk12)
	return err
}
