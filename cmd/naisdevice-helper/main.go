package main

import (
	"context"
	"os"
	"path/filepath"
	"time"

	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/nais/device/internal/helper"
	"github.com/nais/device/internal/helper/config"
	"github.com/nais/device/internal/helper/dns"
	"github.com/nais/device/internal/logger"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/program"
	"github.com/nais/device/internal/unixsocket"
	"github.com/nais/device/internal/version"
	"github.com/nais/device/pkg/pb"
)

var cfg = helper.Config{
	WireGuardConfigPath: filepath.Join(config.ConfigDir, "utun69.conf"),
}

func init() {
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&cfg.Interface, "interface", "utun69", "interface name")

	flag.Parse()
}

func main() {
	log := logger.SetupLogger(cfg.LogLevel, config.LogDir, logger.Helper).WithField("component", "main")

	ctx, cancel := program.MainContext(time.Second)

	otelCancel, err := otel.SetupOTelSDK(ctx, "naisdevice-helper", log)
	if err != nil {
		log.WithError(err).Error("setup OTel SDK")
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := otelCancel(ctx); err != nil {
				log.WithError(err).Error("shutdown OTel SDK")
			}
			cancel()
		}()
	}
	// for windows service control, noop on unix
	err = helper.StartService(log, ctx, cancel)
	if err != nil {
		log.WithError(err).Fatal("starting windows service")
	}

	osConfigurator := helper.NewTracedConfigurator(helper.New(cfg))

	log.WithFields(version.LogFields).WithField("cfg", cfg).Info("starting naisdevice-helper")

	if err := osConfigurator.Prerequisites(); err != nil {
		log.WithError(err).Fatal("checking prerequisites")
	}
	if err := os.MkdirAll(config.RuntimeDir, 0o755); err != nil {
		log.WithError(err).Fatal("setting up runtime dir")
	}
	if err := os.MkdirAll(config.ConfigDir, 0o750); err != nil {
		log.WithError(err).Fatal("setting up config dir")
	}

	unixSocket := filepath.Join(config.RuntimeDir, "helper.sock")
	listener, err := unixsocket.ListenWithFileMode(unixSocket, 0o666)
	if err != nil {
		log.WithError(err).Fatal("failed to listen on unix socket")
	}
	log.WithField("unix_socket", unixSocket).Info("accepting network connections")

	notifier := pb.NewConnectionNotifier()
	grpcServer := grpc.NewServer(grpc.StatsHandler(notifier), grpc.StatsHandler(otel.NewGRPCClientHandler(pb.DeviceHelper_Ping_FullMethodName)))

	dhs := helper.NewDeviceHelperServer(log, cfg, osConfigurator)
	pb.RegisterDeviceHelperServer(grpcServer, dhs)

	teardown := func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second*2)
		defer cancel()
		_, err := dhs.Teardown(ctx, &pb.TeardownRequest{})
		if err != nil {
			log.WithError(err).Warn("teardown failed")
		}
	}

	_, span := otel.Start(ctx, "DNS/Workaround")
	zones := []string{"cloud.nais.io", "intern.nav.no", "intern.dev.nav.no", "knada.io"}
	if err := dns.Apply(zones); err != nil {
		span.RecordError(err)
		log.WithError(err).Error("applying dns config")
	}

	defer teardown()

	go func() {
		for {
			select {
			case <-notifier.Connect():
				log.Info("client gRPC connection established")
			case <-notifier.Disconnect():
				log.Info("client gRPC connection shut down")
				teardown()
			}
		}
	}()

	go func() {
		<-ctx.Done()
		grpcServer.Stop()
	}()

	err = grpcServer.Serve(listener)
	if err != nil {
		log.WithError(err).Fatal("failed to start gRPC server")
	}
	log.Info("gRPC server shut down")
}
