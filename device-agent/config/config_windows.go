package config

import (
	"os"
	"path/filepath"
)

func (c *Config) SetPlatformDefaults() {
	programFiles := os.Getenv("%programfiles%")
	if programFiles == "" {
		programFiles = `c:\Program Files`
	}
	c.WireGuardBinary = filepath.Join(programFiles, "WireGuard", "wireguard.exe")
	c.Interface = "utun69"
}
