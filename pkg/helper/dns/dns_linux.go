package dns

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	configFilePath = "/etc/systemd/resolved.conf.d/99-naisdevice.conf"
)

func apply(zones []string) error {
	err := os.Mkdir(filepath.Dir(configFilePath), 0755)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}

	f, err := os.OpenFile(configFilePath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	err = write(f, zones)
	if err != nil {
		return err
	}

	return reload()
}

func write(out io.Writer, zones []string) error {
	tpl := `[Resolve]
DNS=8.8.8.8#dns.google 8.8.4.4#dns.google 2001:4860:4860::8888#dns.google 2001:4860:4860::8844#dns.google
Domains=%s
DNSOverTLS=opportunistic
`
	prefixedZones := []string{}
	for _, zone := range zones {
		prefixedZones = append(prefixedZones, "~"+zone)
	}

	_, err := fmt.Fprintf(out, tpl, strings.Join(prefixedZones, " "))
	return err
}

func reload() error {
	out, err := exec.Command("systemctl", "restart", "systemd-resolved.service").CombinedOutput()
	if err != nil {
		return fmt.Errorf("reloading systemd-resolved: %w: %s", err, string(out))
	}

	return nil
}
