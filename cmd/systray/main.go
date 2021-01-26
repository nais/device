package main

import (
	"github.com/getlantern/systray"
	"github.com/nais/device/pkg/config"
	"github.com/nais/device/pkg/logger"
	"net"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

type SystrayConfig struct {
	grpcPort uint16
	grpcServer net.IP

	configDir string

	logLevel string
	logFilePath string

	autoConnect bool
}

var(
	cfg = SystrayConfig{
		grpcServer: net.ParseIP("127.0.0.1"),
	}
)

func init() {
	flag.Uint16Var(&cfg.grpcPort, "device-agent-grpc-port", 51801, "port for local (headless) device-agent")
	flag.StringVar(&cfg.logLevel, "log-level", "WARN", "which log level to output")
	flag.BoolVar(&cfg.autoConnect, "connect", false, "auto connect")
	flag.Parse()
}

func main() {
	log.Infof("Starting systray with device-agent %+v", &cfg)
	configDir, err := config.UserConfigDir()
	if err != nil {
		notify("Unable to set up logging: %+v", err)
	}
	cfg.configDir = configDir

	cfg.logFilePath = logger.DeviceAgentLogFilePath(cfg.configDir)
	logger.SetupDeviceLogger(cfg.logLevel, cfg.logFilePath)
	systray.Run(onReady, onExit)
}
