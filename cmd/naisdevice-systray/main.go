package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/nais/device/internal/config"
	"github.com/nais/device/internal/logger"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/systray"
	"github.com/nais/device/internal/version"
)

var log logrus.FieldLogger = logrus.StandardLogger().WithField("component", "main")

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
	ctx, cancel := context.WithCancel(context.Background())
	handleSignals(cancel)
	defer cancel()

	notifier := notify.New(log)
	err := run(ctx, notifier)
	if err != nil {
		notifier.Errorf(err.Error())
		log.Error(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, notifier notify.Notifier) error {
	otelCancel, err := otel.SetupOTelSDK(ctx, "naisdevice-systray", log)
	if err != nil {
		log.WithError(err).Warnf("setup OTel SDK")
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		if err := otelCancel(ctx); err != nil {
			log.WithError(err).Errorf("shutdown OTel SDK")
		}
		cancel()
	}()

	_, span := otel.Start(ctx, "setup")
	defer span.End()

	configDir, err := config.UserConfigDir()
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("start naisdevice-systray: unable to find configuration directory: %w", err)
	}

	// Default config
	cfg := &systray.Config{
		GrpcAddress:        filepath.Join(configDir, "agent.sock"),
		ConfigDir:          configDir,
		LogLevel:           logrus.InfoLevel.String(),
		BlackAndWhiteIcons: false,
	}
	cfg.Populate()
	cfg.Persist()

	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")
	flag.StringVar(&cfg.GrpcAddress, "grpc-address", cfg.GrpcAddress, "path to device-agent unix socket")
	flag.Parse()

	logDir := filepath.Join(cfg.ConfigDir, logger.LogDir)
	log = logger.SetupLogger(cfg.LogLevel, logDir, logger.Systray).WithField("component", "main")
	notifier.SetLogger(log)

	conn, err := net.Dial("unix", cfg.GrpcAddress)
	if err != nil {
		span.AddEvent("agent.start")
		command := exec.CommandContext(ctx, AgentPath)
		err := command.Start()
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("spawning naisdevice-agent: %w", err)
		}
		defer command.Wait()
	} else {
		span.AddEvent("agent.reuse")
		err := conn.Close()
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("closing connection: %w", err)
		}
	}

	log.Infof("naisdevice-systray %s starting up", version.Version)
	log.Infof("configuration: %+v", cfg)

	span.End()
	systray.Spawn(ctx, log.WithField("component", "systray"), *cfg, notifier)
	log.Info("naisdevice-systray shutting down")
	return nil
}
