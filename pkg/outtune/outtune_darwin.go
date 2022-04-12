package outtune

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

const aadLoginURL = "https://nav-no.managed.us2.access-control.cas.ms/aad_login"

type darwin struct {
	helper pb.DeviceHelperClient
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

	// find identities in Mac OS X keychain for this serial
	identities, err := identities(ctx, serial.GetSerial())
	if err != nil {
		return err
	}

	// remove identities
	for _, certificateSerial := range identities {
		cmd := exec.CommandContext(ctx, "/usr/bin/security", "delete-identity", "-Z", certificateSerial, "-t")
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

	id, err := generateKeyAndCertificate(ctx, serial.GetSerial())
	if err != nil {
		return err
	}

	w, err := os.CreateTemp(os.TempDir(), "naisdevice-")
	if err != nil {
		return err
	}
	defer w.Close()
	defer os.Remove(w.Name())

	// Write key+certificate pair to disk in PEM format
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

	currentIdentities, err := identities(ctx, serial.GetSerial())
	if err != nil {
		return fmt.Errorf("unable to find identity in keychain: %s", err)
	}

	cmd = exec.CommandContext(ctx, "/usr/bin/security", "set-identity-preference", "-Z", currentIdentities[0], "-s", aadLoginURL)
	err = cmd.Run()
	if err != nil {
		log.Errorf("set-identity-preference: %s", err)
	}

	return nil
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
