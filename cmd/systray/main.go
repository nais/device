package main

import (
	"context"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nais/device/pkg/config"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/notify"
	"github.com/nais/device/pkg/systray"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func handleSignals(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals

		log.Infof("signal handler: got signal '%s', canceling main context", sig)
		cancel()
		log.Info("signal handler: allowing 1s to stop gracefully...", sig)
		// normally cancelling the context will result in the program returning before the next lines are evaluated
		time.Sleep(1 * time.Second)
		log.Info("signal handler: force-exiting")
		os.Exit(0)
	}()
}

func main() {
	programContext, cancel := context.WithCancel(context.Background())
	handleSignals(cancel)
	defer cancel()

	configDir, err := config.UserConfigDir()
	if err != nil {
		notify.Errorf("start naisdevice-systray: unable to find configuration directory: %v", err)
		os.Exit(1)
	}

	// Default config
	cfg := &systray.Config{
		GrpcAddress:        filepath.Join(configDir, "agent.sock"),
		ConfigDir:          configDir,
		LogLevel:           log.InfoLevel.String(),
		BlackAndWhiteIcons: false,
	}
	cfg.Populate()
	cfg.Persist()

	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")
	flag.StringVar(&cfg.GrpcAddress, "grpc-address", cfg.GrpcAddress, "path to device-agent unix socket")
	flag.Parse()

	logDir := filepath.Join(cfg.ConfigDir, "logs")
	logger.SetupLogger(cfg.LogLevel, logDir, "systray")

	conn, err := net.Dial("unix", cfg.GrpcAddress)
	if err != nil {
		command := exec.CommandContext(programContext, AgentPath)
		err := command.Start()
		if err != nil {
			log.Fatalf("spawning naisdevice-agent: %v", err)
		}
		defer command.Wait()
	} else {
		err := conn.Close()
		if err != nil {
			log.Fatalf("closing connection: %v", err)
		}
	}

	log.Infof("naisdevice-systray %s starting up", version.Version)
	log.Infof("configuration: %+v", cfg)

	systray.Spawn(programContext, *cfg)
	cancel()
	log.Info("naisdevice-systray shutting down")
}
