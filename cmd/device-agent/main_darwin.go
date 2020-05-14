package main

import (
	"github.com/nais/device/device-agent/config"
)

func setPlatform(cfg *config.Config) {
	cfg.Platform = "darwin"
}
