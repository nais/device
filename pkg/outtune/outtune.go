package outtune

import (
	"bufio"
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
	"os"
	"os/exec"
	"regexp"

	"github.com/nais/device/pkg/device-agent/serial"
)

const entropyBits = 4096

type Request struct {
	Serial       string `json:"serial"`
	PublicKeyPEM string `json:"public_key_pem"`
}

type Response struct {
	CertificatePEM string `json:"cert_pem"`
}

func Purge(ctx context.Context) error {
	ser, err := serial.GetDeviceSerial("")
	if err != nil {
		return err
	}

	ids, err := identities(ctx, ser)
	if err != nil {
		return err
	}

	for _, id := range ids {
		fmt.Println(id)
	}

	return nil
}

func GetCertificate(ctx context.Context) error {
	ser, err := serial.GetDeviceSerial("")
	if err != nil {
		return err
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, entropyBits)
	if err != nil {
		return err
	}

	cert, err := download(ctx, ser, privateKey)
	if err != nil {
		return err
	}

	w, err := os.CreateTemp(os.TempDir(), "foobar")
	if err != nil {
		return err
	}
	defer w.Close()
	defer os.Remove(w.Name())

	err = pem.Encode(w, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	if err != nil {
		return err
	}

	_, err = w.WriteString(cert.CertificatePEM)
	if err != nil {
		return err
	}

	// private key and certificate written; flush contents to disk and close
	err = w.Close()
	if err != nil {
		return err
	}

	// run Mac OS X keychain import tool
	cmd := exec.CommandContext(ctx, "/usr/bin/security", "import", w.Name(), "-A")
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
func download(ctx context.Context, serial string, privateKey *rsa.PrivateKey) (*Response, error) {
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

	req := &Request{
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

	response := &Response{}
	err = json.NewDecoder(resp.Body).Decode(response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func identities(ctx context.Context, serial string) ([]string, error) {
	id := "naisdevice - " + serial
	cmd := exec.CommandContext(ctx, "/usr/bin/security", "find-identity", "-s", id)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer stdout.Close()

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0)
	re := regexp.MustCompile("[A-Za-z0-9]{40}")
	scan := bufio.NewScanner(stdout)
	for scan.Scan() {
		line := scan.Text()
		certificateID := re.FindString(line)
		if len(certificateID) == 0 {
			continue
		}
		ids = append(ids, certificateID)
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	return ids, nil
}

//pubkey_path="$HOME/Library/Application Support/naisdevice/browser_cert_pubkey.pem"
//    set -eo pipefail
//    cd "$(mktemp -d)"
//    ## delete expired cert
//    security delete-certificate -c "$cn"
//
//    ## renew cert and import in keychain
//    download_cert
//    security import cert.pem
//    identity_cert=$(security find-certificate -c "$cn" -Z | grep "SHA-1 hash:")
//    certhash=$(echo "$identity_cert" | cut -c13-53)
//
//    ## set identity preference to use this cert automaticlaly for specified domains
//    security set-identity-preference -Z "$certhash" -s "https://nav-no.managed.us2.access-control.cas.ms/aad_login"
//  ) || (echo "failed renewing cert"; exit 1)
