package outtune

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

const entropyBits = 4096
const dummyPassword = "asd123"

type Outtune interface {
	Install(ctx context.Context) error
}

type request struct {
	Serial       string `json:"serial"`
	PublicKeyPEM string `json:"public_key_pem"`
}

type response struct {
	CertificatePEM string `json:"cert_pem"`
}

func generateKeyAndCertificate(ctx context.Context, serial string) (*identity, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, entropyBits)
	if err != nil {
		return nil, err
	}

	resp, err := download(ctx, serial, privateKey)
	if err != nil {
		return nil, err
	}

	block, rest := pem.Decode([]byte(resp.CertificatePEM))
	if len(rest) > 0 {
		log.Warnf("certificate had remaining input which was ignored")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &identity{
		privateKey:  privateKey,
		certificate: cert,
	}, nil
}

func download(ctx context.Context, serial string, privateKey *rsa.PrivateKey) (*response, error) {
	publicKey, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKey,
	}

	buf := &bytes.Buffer{}
	err = pem.Encode(buf, block)
	if err != nil {
		return nil, fmt.Errorf("encode public key in PEM format: %w", err)
	}

	req := &request{
		Serial:       serial,
		PublicKeyPEM: base64.StdEncoding.EncodeToString(buf.Bytes()),
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request data: %w", err)
	}

	const url = "https://outtune-api.prod-gcp.nais.io/local/cert"
	httpRequest, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}

	resp, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("send request to CA signer: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	default:
		msg, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("CA signer returned %s: %s", resp.Status, string(msg))
	}

	response := &response{}
	err = json.NewDecoder(resp.Body).Decode(response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
