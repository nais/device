package serial

import (
	"fmt"
	"os/exec"
	"strings"
)

func GetDeviceSerial() (string, error) {
	cmd := exec.Command("wmic", "bios", "get", "serialnumber")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("getting serial with ioreg: %w: %v", err, string(b))
	}

	lines := strings.Split(string(b), "\n")
	return lines[1], nil
}
