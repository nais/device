package outtune

import (
	"context"
	"crypto/x509"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/pb"
)

const (
	defaultNSSPath             = ".pki/nssdb"
	firefoxProfilesGlob        = ".mozilla/firefox/*.default-release*"
	firefoxSnapProfilesGlob    = "snap/firefox/common/.mozilla/firefox/*.default-release"
	certutilBinary             = "/usr/bin/certutil"
	pk12utilBinary             = "/usr/bin/pk12util"
	naisdeviceCertName         = "naisdevice"
	clientAuthRememberListFile = "ClientAuthRememberList.txt"
)

type linux struct {
	helper pb.DeviceHelperClient
}

func New(helper pb.DeviceHelperClient) Outtune {
	return &linux{
		helper: helper,
	}
}

func (o *linux) Cleanup(ctx context.Context) error {
	dbs, err := nssDatabases()
	if err != nil {
		return err
	}

	for _, db := range dbs {
		oldCertificates, err := listNaisdeviceCertificates(ctx, db)
		if err != nil {
			log.Warnf("outtune: list certificates in db %s: %v", db, err)
		}

		for _, certificate := range oldCertificates {
			err = deleteCertificate(ctx, db, certificate)
			if err != nil {
				log.Infof("outtune: delete certificate '%s' in db %s: %v", certificate, db, err)
			}
		}
		orphanedKeys, err := listNaisdeviceKeys(ctx, db)
		if err != nil {
			log.Warnf("outtune: list keys in db %s: %v", db, err)
		}

		// Delete remaining keys (remains from old buggy code)
		for _, orphanedKey := range orphanedKeys {
			err = deleteKey(ctx, db, orphanedKey)
			if err != nil {
				log.Warnf("outtune: delete key '%s' in db %s: %v", orphanedKey, db, err)
			}
		}
	}

	return nil
}

func (o *linux) Install(ctx context.Context) error {
	// We have to delete before install as all certs have the same nickname, and certutil can only delete by nickname :sadkek:
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
		err = installCert(ctx, db, w.Name())
		if err != nil {
			return err
		}

		err = persistClientAuthRememberList(db, id.certificate)
		if err != nil {
			return err
		}

		certificates, err := listNaisdeviceCertificates(ctx, db)
		if err != nil {
			log.Warnf("outtune: list certificates in db %s: %v", db, err)
		}

		if len(certificates) > 1 {
			log.Warnf("outtune: BUG: more than 1 naisdevice certificate present is %s!\n%#v", db, certificates)
		}
	}

	return nil
}

func (o *linux) Expired(ctx context.Context) (bool, error) {
	serial, err := o.helper.GetSerial(ctx, &pb.GetSerialRequest{})
	if err != nil {
		return false, err
	}

	certname := fmt.Sprintf("naisdevice - %v - NAV", serial.GetSerial())

	dbs, err := nssDatabases()
	if err != nil {
		return false, err
	}

	expiredAfter := time.Now().Add(time.Hour * 24 * 10).UTC()

	for _, db := range dbs {
		cmd := exec.CommandContext(ctx, certutilBinary, "-d", db, "-V", "-u", "A", "-n", certname, "-b", expiredAfter.Format("060102150405Z"))
		if err := cmd.Run(); err != nil {
			return true, err
		}
	}

	return false, nil
}

func deleteCertificate(ctx context.Context, db, certname string) error {
	cmd := exec.CommandContext(ctx, certutilBinary, "-d", db, "-F", "-n", certname)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete cert: %w: %s", err, string(out))
	}

	return nil
}

func listNaisdeviceCertificates(ctx context.Context, db string) ([]string, error) {
	cmd := exec.CommandContext(ctx, certutilBinary, "-d", db, "-L")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(out))
	}

	lines := strings.Split(string(out), "\n")
	certificateName := regexp.MustCompile(`^(naisdevice\s-\s[^\s]+\s-\sNAV|naisdevice).*$`)
	var ret []string
	for _, line := range lines {
		match := certificateName.FindStringSubmatch(line)
		if len(match) == 2 {
			ret = append(ret, match[1])
		}
	}

	return ret, nil
}

func installCert(ctx context.Context, db, pk12filename string) error {
	cmd := exec.CommandContext(ctx, pk12utilBinary, "-d", db, "-i", pk12filename, "-W", dummyPassword)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install cert: %w: %s", err, string(out))
	}
	return nil
}

func persistClientAuthRememberList(db string, cert *x509.Certificate) error {
	dbkey, err := GenerateDBKey(cert)
	if err != nil {
		return err
	}

	rememberList := GenerateClientAuthRememberList(dbkey)
	filename := fmt.Sprintf("%s/%s", db, clientAuthRememberListFile)

	var file *os.File
	_, err = os.Stat(filename)
	if os.IsNotExist(err) {
		file, err = os.OpenFile(filename, os.O_EXCL|os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	} else {
		// todo: consider reading the file in order to identify matching rememberlist-entries
		// and replace them with our new entries
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_EXCL, 0o644)
	}
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write([]byte(rememberList))
	if err != nil {
		return err
	}

	return file.Close()
}

func nssDatabases() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine user home directory: %v", err)
	}

	var nssDBs []string
	_, err = os.Stat(fmt.Sprintf("%s/%s", home, defaultNSSPath))
	if err == nil {
		nssDBs = append(nssDBs, fmt.Sprintf("%s/%s", home, defaultNSSPath))
	} else {
		log.Infof("could not find default nss path: %v", err)
	}

	firefoxProfiles, err := filepath.Glob(fmt.Sprintf("%s/%s", home, firefoxProfilesGlob))
	if err != nil {
		log.Infof("could not find any firefox profiles: %v", err)
	}
	nssDBs = append(nssDBs, firefoxProfiles...)

	firefoxSnapProfiles, err := filepath.Glob(fmt.Sprintf("%s/%s", home, firefoxSnapProfilesGlob))
	if err != nil {
		log.Infof("could not find any firefox snap profiles: %v", err)
	}

	nssDBs = append(nssDBs, firefoxSnapProfiles...)

	return nssDBs, nil
}

func deleteKey(ctx context.Context, db, keyId string) error {
	cmd := exec.CommandContext(ctx, certutilBinary, "-d", db, "-F", "-k", keyId)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("delete cert: %w: %s", err, string(out))
	}

	return nil
}

func listNaisdeviceKeys(ctx context.Context, db string) ([]string, error) {
	cmd := exec.CommandContext(ctx, certutilBinary, "-d", db, "-K")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, string(out))
	}

	lines := strings.Split(string(out), "\n")
	naisdeviceKeyId := regexp.MustCompile(`^<\s*\d+>\s*rsa\s*([^\s]+)\s*naisdevice.*$`)
	var ret []string
	for _, line := range lines {
		match := naisdeviceKeyId.FindStringSubmatch(line)
		if len(match) == 2 {
			ret = append(ret, match[1])
		}
	}

	return ret, nil
}
