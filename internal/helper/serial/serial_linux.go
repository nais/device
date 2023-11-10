package serial

import (
	"fmt"
	"os"
	"strings"
)

const productSerialPath = "/sys/devices/virtual/dmi/id/product_serial"

func GetDeviceSerial() (string, error) {
	serial, err := os.ReadFile(productSerialPath)
	if err != nil {
		return "", fmt.Errorf("reading product serial from disk: %w", err)
	}
	return strings.TrimSuffix(string(serial), "\n"), nil
}
