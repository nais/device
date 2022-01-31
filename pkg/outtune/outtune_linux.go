package outtune

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

const (
	defaultNSSPath      = ".pki/nssdb"
	firefoxProfilesGlob = ".mozilla/firefox/*.default-release*"
	certutilBinary      = "/usr/bin/certutil"
	pk12utilBinary      = "/usr/bin/pk12util"
	naisdeviceCertName  = "naisdevice"
)

type linux struct {
	helper pb.DeviceHelperClient
}

func New(helper pb.DeviceHelperClient) Outtune {
	return &linux{
		helper: helper,
	}
}

func (o *linux) Install(ctx context.Context) error {
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
	err = id.SerializePKCS12(w)
	if err != nil {
		return err
	}

	// flush contents to disk
	err = w.Close()
	if err != nil {
		return err
	}

	dbs, err := nssDatabases()
	if err != nil {
		return err
	}
	for _, db := range dbs {
		certs, err := listCertificates(ctx, db)
		if err != nil {
			log.Infof("could not list certificates in db %s: %v", db, err)
		}
		err = installCert(ctx, db, w.Name())
		if err != nil {
			return err
		}
		for _, cert := range certs {
			err = deleteCert(ctx, db, cert) // this seems to always fail, but the cert does get deleted?
			if err != nil {
				log.Infof("couldn't delete cert '%s' in db %s: %v", cert, db, err)
			}
		}
	}
	return nil
}

func deleteCert(ctx context.Context, db, certname string) error {
	cmd := exec.CommandContext(ctx, certutilBinary, "-d", db, "-F", "-n", certname)
	return cmd.Run()
}

func listCertificates(ctx context.Context, db string) ([]string, error) {
	cmd := exec.CommandContext(ctx, certutilBinary, "-d", db, "-L")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")
	var ret []string
	for _, line := range lines {
		if strings.HasPrefix(line, naisdeviceCertName) {
			ret = append(ret, strings.TrimSpace(strings.TrimSuffix(line, "u,u,u")))
		}
	}

	return ret, nil
}

func installCert(ctx context.Context, db, pk12filename string) error {
	cmd := exec.CommandContext(ctx, pk12utilBinary, "-d", db, "-i", pk12filename, "-W", dummyPassword)
	err := cmd.Run()
	return err
}

func nssDatabases() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine user home directory: %v", err)
	}

	var nss_dbs []string
	_, err = os.Stat(fmt.Sprintf("%s/%s", home, defaultNSSPath))
	if err == nil {
		nss_dbs = append(nss_dbs, fmt.Sprintf("%s/%s", home, defaultNSSPath))
	} else {
		log.Infof("could not find default nss path: %v", err)
	}

	firefoxProfiles, err := filepath.Glob(fmt.Sprintf("%s/%s", home, firefoxProfilesGlob))
	if err != nil {
		log.Infof("could not find any firefox profiles: %v", err)
	}
	nss_dbs = append(nss_dbs, firefoxProfiles...)
	return nss_dbs, nil
}
