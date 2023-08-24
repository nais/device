package dns

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

const configFileDir = "/etc/resolved"

func apply(zones []string) error {
	err := os.Mkdir(configFileDir, 0755)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}

	for _, zone := range zones {
		configFilePath := filepath.Join(configFileDir, fmt.Sprintf("%s.conf", zone))
		f, err := os.OpenFile(configFilePath, os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		err = write(f)
		if err != nil {
			return err
		}
	}

	return reload()
}

func write(out io.Writer) error {
	tpl := `nameserver 8.8.8.8
nameserver 8.8.4.4`
	_, err := io.WriteString(out, tpl)
	return err
}

func reload() error {
	out, err := exec.Command("sudo", "killall", "-HUP", "mDNSResponder").CombinedOutput()
	if err != nil {
		return fmt.Errorf("reloading mDNSResponder config: %w: %s", err, string(out))
	}

	return nil
}