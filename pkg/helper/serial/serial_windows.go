package serial

import (
	"bytes"
	"fmt"
	"os/exec"
)

func GetDeviceSerial() (string, error) {
	cmd := exec.Command("wmic", "bios", "get", "serialnumber")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("getting serial with wmic: %w: %v", err, string(b))
	}

	lines := bytes.Split(b, []byte("\r\r\n"))
	trimmed := bytes.TrimSpace(lines[1])
	return string(trimmed), nil
}
