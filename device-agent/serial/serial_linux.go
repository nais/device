package serial

import (
	"fmt"
	"io/ioutil"
	"strings"
)

func GetDeviceSerial(serialpath string) (string, error) {
	serial, err := ioutil.ReadFile(serialpath)
	if err != nil {
		return "", fmt.Errorf("reading product serial from disk: %w", err)
	}
	return strings.TrimSuffix(string(serial), "\n"), nil
}
