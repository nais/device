package config

import (
	"path/filepath"
)

func (c *Config) SetPlatformDefaults() {
	c.WireGuardBinary = filepath.Join(c.BinaryDir, "naisdevice-wg")
	c.WireGuardGoBinary = filepath.Join(c.BinaryDir, "naisdevice-wireguard-go")
	c.Interface = "wg0"
}
