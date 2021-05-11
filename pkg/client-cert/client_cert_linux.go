package client_cert

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

const (
	importCertScriptPath = "/usr/share/naisdevice/import_cert.sh"
)

func Renew() error {
	output, err := exec.Command("/bin/bash", "-c", importCertScriptPath).CombinedOutput()
	if err == nil {
		return nil
	}
	log.Errorf("executing cert renewal script: %s", string(output))
	return fmt.Errorf("executing cert renewal script: %v", err)
}
