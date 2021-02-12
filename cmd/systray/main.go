package main

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nais/device/pkg/config"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/notify"
	"github.com/nais/device/pkg/systray"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func main() {
	configDir, err := config.UserConfigDir()
	if err != nil {
		notify.Errorf("start naisdevice-systray: unable to find configuration directory: %v", err)
		os.Exit(1)
	}

	cfg := systray.Config{
		GrpcAddress: filepath.Join(configDir, "agent.sock"),
		ConfigDir:   configDir,
		LogLevel:    log.InfoLevel.String(),
		AutoConnect: false,
	}
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")
	flag.BoolVar(&cfg.AutoConnect, "connect", cfg.AutoConnect, "auto connect")
	flag.StringVar(&cfg.GrpcAddress, "grpc-address", cfg.GrpcAddress, "path to device-agent unix socket")
	flag.Parse()

	logger.SetupLogger(cfg.LogLevel, cfg.ConfigDir, "systray.log")

	conn, err := net.Dial("unix", cfg.GrpcAddress)
	if err != nil {
		// TODO: remove when agent runs as service
		ctx, cancel := context.WithCancel(context.Background())
		err = exec.CommandContext(ctx, AgentPath).Start()
		if err != nil {
			log.Fatalf("spawning naisdevice-agent: %v", err)
		}
		defer cancel()
	} else {
		err := conn.Close()
		if err != nil {
			log.Fatalf("closing connection: %v", err)
		}
	}

	log.Infof("naisdevice-systray %s starting up", version.Version)
	log.Infof("configuration: %+v", cfg)

	systray.Spawn(cfg)
}
