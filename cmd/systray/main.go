package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gen2brain/beeep"
	"github.com/nais/device/pkg/config"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/systray"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func main() {
	configDir, err := config.UserConfigDir()
	if err != nil {
		notify("Can't start naisdevice: unable to find configuration directory: %v", err)
		os.Exit(1)
	}

	cfg := systray.Config{
		GrpcAddress: filepath.Join(configDir, "agent.sock"),
		ConfigDir:   configDir,
	}

	logger.SetupLogger(cfg.LogLevel, cfg.ConfigDir, "systray.log")

	flag.StringVar(&cfg.LogLevel, "log-level", "warning", "which log level to output")
	flag.BoolVar(&cfg.AutoConnect, "connect", false, "auto connect")
	flag.StringVar(&cfg.GrpcAddress, "grpc-address", cfg.GrpcAddress, "path to device-agent unix socket")
	flag.Parse()

	cfg.ConfigDir = configDir

	log.Infof("Starting systray with device-agent %+v", &cfg)

	systray.Spawn(cfg)
}

func notify(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	err := beeep.Notify("NAIS device", message, "../Resources/nais-logo-red.png")
	log.Infof("sending message to notification centre: %s", message)
	if err != nil {
		log.Errorf("failed sending message due to error: %s", err)
	}
}
