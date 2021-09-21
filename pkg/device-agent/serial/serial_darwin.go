package serial

import (
	"fmt"
	"os/exec"
	"regexp"
)

func GetDeviceSerial(serialpath string) (string, error) {
	cmd := exec.Command("/usr/sbin/ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("getting serial with ioreg: %w: %v", err, string(b))
	}

	re := regexp.MustCompile("\"IOPlatformSerialNumber\" = \"([^\"]+)\"")
	matches := re.FindSubmatch(b)

	if len(matches) != 2 {
		return "", fmt.Errorf("unable to extract serial from output: %v", string(b))
	}

	return string(matches[1]), nil
}
