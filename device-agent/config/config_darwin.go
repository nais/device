package config

import (
	"path/filepath"
)

const (
	BinaryDir ="/Applications/naisdevice.app/Contents/MacOS"
)

func (c *Config) SetPlatformDefaults() {
	c.WireGuardBinary = filepath.Join(BinaryDir, "wg")
	c.WireGuardGoBinary = filepath.Join(BinaryDir, "wireguard-go")
	c.Interface = "utun69"
}
