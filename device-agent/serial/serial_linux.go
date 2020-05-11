package serial

import (
	"fmt"
	"os/exec"
	"regexp"
)

func GetDeviceSerial() (string, error) {
	cmd := exec.Command("/usr/bin/sudo", "dmidecode", "-t", "system")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("getting serial with dmidecode: %w: %v", err, string(b))
	}

	r := regexp.MustCompile(`Serial Number: (.+)\n`)
	matches := r.FindStringSubmatch(string(b))

	if len(matches) != 2 {
		return "", fmt.Errorf("unable to extract serial from output: %v", string(b))
	}

	return matches[1], nil
}
