package config

import (
	"path/filepath"
)

const (
	BinaryDir = "/usr/bin/"
)

func (c *Config) SetPlatformDefaults() {
	c.WireGuardBinary = filepath.Join(BinaryDir, "wg")
	c.Interface = "wg0"
}
