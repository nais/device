package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/nais/device/internal/logger"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/program"
	"github.com/nais/device/internal/systray"
	"github.com/nais/device/internal/version"
	"github.com/nais/device/pkg/config"
)

var log logrus.FieldLogger = logrus.StandardLogger().WithField("component", "main")

func main() {
	ctx, cancel := program.MainContext(time.Second)
	defer cancel()

	notifier := notify.New(log)
	err := run(ctx, notifier)
	if err != nil {
		notifier.Errorf(err.Error())
		log.WithError(err).Error("run error")
		os.Exit(1)
	}
}

func run(ctx context.Context, notifier notify.Notifier) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	otelCancel, err := otel.SetupOTelSDK(ctx, "naisdevice-systray", log)
	if err != nil {
		log.WithError(err).Warn("setup OTel SDK")
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		if err := otelCancel(ctx); err != nil {
			log.WithError(err).Error("shutdown OTel SDK")
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
		defer func() {
			cancel()
			err := command.Wait()
			if err != nil {
				log.WithError(err).Error("naisdevice-agent exited")
			}
		}()
	} else {
		span.AddEvent("agent.reuse")
		err := conn.Close()
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("closing connection: %w", err)
		}
	}

	log.WithFields(version.LogFields).WithField("cfg", cfg).Info("starting naisdevice-systray")

	span.End()
	systray.Spawn(ctx, log.WithField("component", "systray"), *cfg, notifier)
	log.Info("naisdevice-systray shutting down")
	return nil
}
