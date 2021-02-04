package main

import (
	"fmt"
	"github.com/gen2brain/beeep"
	"github.com/nais/device/pkg/config"
	"github.com/nais/device/pkg/logger"
	"net"

	"github.com/nais/device/pkg/systray"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func main() {
	cfg := systray.Config{
		GrpcServer: net.ParseIP("127.0.0.1"),
	}

	flag.Uint16Var(&cfg.GrpcPort, "device-agent-grpc-port", 51801, "port for local (headless) device-agent")
	flag.StringVar(&cfg.LogLevel, "log-level", "warning", "which log level to output")
	flag.BoolVar(&cfg.AutoConnect, "connect", false, "auto connect")
	flag.Parse()

	configDir, err := config.UserConfigDir()
	if err != nil {
		notify("Unable to set up logging: %+v", err)
	}

	cfg.ConfigDir = configDir

	cfg.LogFilePath = logger.DeviceAgentLogFilePath(cfg.ConfigDir)
	logger.SetupDeviceLogger(cfg.LogLevel, cfg.LogFilePath)
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
