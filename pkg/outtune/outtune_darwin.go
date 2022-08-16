package outtune

import (
	"bufio"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

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
	certPem string `json:"cert_pem"`
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

	certificates(ctx, serial.GetSerial())

	// find identities in Mac OS X keychain for this serial
	identities, err := identities(ctx, serial.GetSerial())
	if err != nil {
		return err
	}

	// remove identities
	for _, certificateSerial := range identities {
		cmd := exec.CommandContext(ctx, "/usr/bin/security", "delete-certificate", "-Z", certificateSerial, "-t")
		err = cmd.Run()
		if err != nil {
			log.Errorf("unable to delete certificate and private key from keychain: %s", err)
		} else {
			log.Debugf("deleted identity '%s' from keychain", certificateSerial)
		}
	}

	return nil
}

func (o *darwin) Install(ctx context.Context) error {
	o.Cleanup(ctx)

	serial, err := o.helper.GetSerial(ctx, &pb.GetSerialRequest{})
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine user home directory: %v", err)
	}

	pubKeyPath := filepath.Join(home, certPath)

	pubKey, err := os.OpenFile(pubKeyPath, os.O_RDONLY, 0o644)
	if errors.Is(err, fs.ErrNotExist) {
		err := o.generate(ctx, serial.GetSerial(), pubKeyPath)
		if err != nil {
			return err
		}
	} else {
		defer pubKey.Close()

		req, err := http.NewRequestWithContext(ctx, "POST", "https://outtune-api.prod-gcp.nais.io/local/cert", pubKey)
		if err != nil {
			return err
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		certResponse := &certresponse{}
		err = json.Unmarshal(body, certResponse)
		if err != nil {
			return err
		}
		tempCertFile, err := os.CreateTemp(os.TempDir(), "naisdevice-cert-")
		if err != nil {
			return err
		}
		defer tempCertFile.Close()
		defer os.Remove(tempCertFile.Name())
		_, err = tempCertFile.WriteString(certResponse.certPem)
		if err != nil {
			return err
		}
		// flush contents to disk
		err = tempCertFile.Close()
		if err != nil {
			return err
		}
		cmd := exec.CommandContext(ctx, "/usr/bin/security", "import", tempCertFile.Name())
		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	currentIdentities, err := identities(ctx, serial.GetSerial())
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

func (o *darwin) generate(ctx context.Context, serial string, pubKeyPath string) error {
	id, err := generateKeyAndCertificate(ctx, serial)
	if err != nil {
		return err
	}
	w, err := os.CreateTemp(os.TempDir(), "naisdevice-")
	if err != nil {
		return err
	}
	defer w.Close()
	defer os.Remove(w.Name())
	// Write key+certificate pair to disk
	err = id.SerializePEM(w)
	if err != nil {
		return err
	}
	// flush contents to disk
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

	publicKey, err := x509.MarshalPKIXPublicKey(&id.privateKey.PublicKey)
	if err != nil {
		return err
	}

	return os.WriteFile(pubKeyPath, publicKey, 0o644)
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

	idMap := make(map[string]struct{})
	re := regexp.MustCompile("[A-Za-z0-9]{40}")
	scan := bufio.NewScanner(stdout)
	for scan.Scan() {
		line := scan.Text()
		certificateID := re.FindString(line)
		if len(certificateID) == 0 || !strings.Contains(line, "naisdevice") {
			continue
		}
		idMap[certificateID] = struct{}{}
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
	re := regexp.MustCompile(`SHA-1 hash:\s[A-Za-z0-9]{40}`)
	scan := bufio.NewScanner(stdout)
	for scan.Scan() {
		line := scan.Text()
		matches := re.FindAllStringSubmatch(line, 1)
		if len(matches) < 2 {
			continue
		}
		idMap[matches[1][1]] = struct{}{}
	}

	err = cmd.Wait()
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for id := range idMap {
		fmt.Println("######### FOUND ", id)
		ids = append(ids, id)
	}

	return ids, nil
}
