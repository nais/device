package filesystem

import (
	"fmt"
	"os"

	"github.com/nais/device/pkg/device-agent/config"
)

func EnsurePrerequisites(c *config.Config) error {
	if err := filesExist(c.WireGuardBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %v", err)
	}

	if err := ensureDirectories(c.ConfigDir); err != nil {
		return fmt.Errorf("ensuring directory exists: %v", err)
	}

	return ensurePlatformPrerequisites(c)
}

func FileMustExist(filepath string) error {
	info, err := os.Stat(filepath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%v is a directory", filepath)
	}

	return nil
}

func filesExist(files ...string) error {
	for _, file := range files {
		if err := FileMustExist(file); err != nil {
			return err
		}
	}

	return nil
}

func ensureDirectories(dirs ...string) error {
	for _, dir := range dirs {
		if err := ensureDirectory(dir); err != nil {
			return err
		}
	}

	return nil
}

func ensureDirectory(dir string) error {
	info, err := os.Stat(dir)

	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o700)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%v is a file", dir)
	}

	return nil
}
