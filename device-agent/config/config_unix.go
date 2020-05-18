// +build linux darwin

package config

import (
	"fmt"
	"path/filepath"
)

func (c *Config) SetPlatformDefaults() {
	c.WireGuardBinary = filepath.Join(c.BinaryDir, "naisdevice-wg")
	c.WireGuardGoBinary = filepath.Join(c.BinaryDir, "naisdevice-wireguard-go")
}

func (c *Config) EnsurePlatformPrerequisites() error {
	if err := ensureDirectories(c.BinaryDir); err != nil {
		return fmt.Errorf("ensuring directory exists: %w", err)
	}

	if err := filesExist(c.WireGuardGoBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}

	return nil
}
