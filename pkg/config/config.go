package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

func UserConfigDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		var dir string

		dir = os.Getenv("PROGRAMDATA")
		if dir == "" {
			return "", errors.New("%PROGRAMDATA% is not defined")
		}
		dir += "\\NAV\\naisdevice"

		return dir, nil

	default:
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		} else {
			return filepath.Join(userConfigDir, "naisdevice"), err
		}
	}
}
