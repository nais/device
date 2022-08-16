package outtune

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

const (
	aadLoginURL = "https://nav-no.managed.us2.access-control.cas.ms/aad_login"
	certPath    = "Library/Application Support/naisdevice/browser_cert_pubkey.pem"
)

type darwin struct {
	helper pb.DeviceHelperClient
}

type certresponse struct {
	CertPem string `json:"cert_pem"`
}

func New(helper pb.DeviceHelperClient) Outtune {
	return &darwin{
		helper: helper,
	}
}

// Cleans up all naisdevice-certificates and keys for the given serial.
func (o *darwin) Cleanup(ctx context.Context) error {
	serial, err := o.helper.GetSerial(ctx, &pb.GetSerialRequest{})
	if err != nil {
		return err
	}

	// find certificates in Mac OS X keychain for this serial
	certificates, err := certificates(ctx, serial.GetSerial())
	if err != nil {
		return err
	}

	// remove identities
	for _, certificateSerial := range certificates {
		cmd := exec.CommandContext(ctx, "/usr/bin/security", "delete-certificate", "-Z", certificateSerial)
		err = cmd.Run()
		if err != nil {
			log.Errorf("unable to delete certificate and private key from keychain: %s", err)
		} else {
			log.Debugf("deleted certificate '%s' from keychain", certificateSerial)
		}
	}

	return nil
}

func (o *darwin) Install(ctx context.Context) error {
	serial, err := o.helper.GetSerial(ctx, &pb.GetSerialRequest{})
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine user home directory: %v", err)
	}

	pubKeyPath := filepath.Join(home, certPath)

	var pk *rsa.PrivateKey
	_, err = os.Stat(pubKeyPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		pk, err = o.generate(ctx, serial.GetSerial(), pubKeyPath)
		if err != nil {
			return err
		}
	} else {
		o.Cleanup(ctx)
	}

	resp, err := o.download_cert(ctx, serial.GetSerial(), pubKeyPath)
	if err != nil {
		return err
	}
	err = o.importIdentity(ctx, pk, resp.CertificatePEM)
	if err != nil {
		return err
	}

	currentIdentities, err := certificates(ctx, serial.GetSerial())
	if err != nil || len(currentIdentities) == 0 {
		return fmt.Errorf("unable to find identity in keychain: %s", err)
	}

	cmd := exec.CommandContext(ctx, "/usr/bin/security", "set-identity-preference", "-Z", currentIdentities[0], "-s", aadLoginURL)
	err = cmd.Run()
	if err != nil {
		log.Errorf("set-identity-preference: %s", err)
	}

	return nil
}

func (o *darwin) download_cert(ctx context.Context, serial, pubKeyPath string) (*response, error) {
	publicKey, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return nil, err
	}

	return downloadWithPublicKey(ctx, serial, publicKey)
}

func (o *darwin) generate(ctx context.Context, serial, pubKeyPath string) (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, entropyBits)
	if err != nil {
		return nil, err
	}

	publicKey, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return privateKey, os.WriteFile(pubKeyPath, publicKey, 0o644)
}

func (o *darwin) importIdentity(ctx context.Context, privateKey *rsa.PrivateKey, certificate string) error {
	w, err := os.CreateTemp(os.TempDir(), "naisdevice-")
	if err != nil {
		return err
	}
	defer w.Close()
	defer os.Remove(w.Name())

	if privateKey != nil {
		pem.Encode(w, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		})
	}

	block, rest := pem.Decode([]byte(certificate))
	if len(rest) > 0 {
		log.Warnf("certificate had remaining input which was ignored")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	pem.Encode(w, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	// flush contents to disk
	err = w.Close()
	if err != nil {
		return err
	}
	// run Mac OS X keychain import tool
	cmd := exec.CommandContext(ctx, "/usr/bin/security", "import", w.Name(), "-A")
	return cmd.Run()
}

func certificates(ctx context.Context, serial string) ([]string, error) {
	id := "naisdevice - " + serial
	cmd := exec.CommandContext(ctx, "/usr/bin/security", "find-certificate", "-c", id, "-Z")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer stdout.Close()

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	idMap := make(map[string]struct{})
	re := regexp.MustCompile(`SHA-1 hash:\s([A-Za-z0-9]{40})`)
	scan := bufio.NewScanner(stdout)
	for scan.Scan() {
		line := scan.Text()
		matches := re.FindAllStringSubmatch(line, 1)
		if len(matches) == 0 {
			continue
		}
		idMap[matches[0][1]] = struct{}{}
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for id := range idMap {
		ids = append(ids, id)
	}

	return ids, nil
}
