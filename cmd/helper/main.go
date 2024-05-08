package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	flag "github.com/spf13/pflag"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"

	"github.com/nais/device/internal/helper"
	"github.com/nais/device/internal/helper/config"
	"github.com/nais/device/internal/helper/dns"
	"github.com/nais/device/internal/logger"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/unixsocket"
	"github.com/nais/device/internal/version"
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

	programContext, cancel := context.WithCancel(context.Background())
	otelCancel, err := otel.SetupOTelSDK(programContext, "naisdevice-helper", log)
	if err != nil {
		log.Fatalf("setup OTel SDK: %s", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := otelCancel(ctx); err != nil {
			log.Errorf("shutdown OTel SDK: %s", err)
		}
		cancel()
	}()

	// for windows service control, noop on unix
	err = helper.StartService(log, programContext, cancel)
	if err != nil {
		log.Fatalf("Starting windows service: %v", err)
	}

	osConfigurator := &helper.TracedConfigurator{Wrapped: helper.New(cfg)}

	log.Infof("naisdevice-helper %s starting up", version.Version)
	log.Infof("configuration: %+v", cfg)

	if err := osConfigurator.Prerequisites(); err != nil {
		log.Fatalf("Checking prerequisites: %v", err)
	}
	if err := os.MkdirAll(config.RuntimeDir, 0o755); err != nil {
		log.Fatalf("Setting up runtime dir: %v", err)
	}
	if err := os.MkdirAll(config.ConfigDir, 0o750); err != nil {
		log.Fatalf("Setting up config dir: %v", err)
	}

	grpcPath := filepath.Join(config.RuntimeDir, "helper.sock")
	listener, err := unixsocket.ListenWithFileMode(grpcPath, 0o666)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("accepting network connections on unix socket %s", grpcPath)

	notifier := pb.NewConnectionNotifier()
	grpcServer := grpc.NewServer(grpc.StatsHandler(notifier), grpc.StatsHandler(otelgrpc.NewServerHandler()))

	dhs := helper.NewDeviceHelperServer(log, cfg, osConfigurator)
	pb.RegisterDeviceHelperServer(grpcServer, dhs)

	teardown := func() {
		ctx, cancel := context.WithTimeout(programContext, time.Second*2)
		defer cancel()
		_, err := dhs.Teardown(ctx, &pb.TeardownRequest{})
		if err != nil {
			log.Warn(err)
		}
	}

	_, span := otel.Start(programContext, "DNS/Workaround")
	zones := []string{"cloud.nais.io", "intern.nav.no", "intern.dev.nav.no", "knada.io"}
	if err := dns.Apply(zones); err != nil {
		span.RecordError(err)
		log.Errorf("applying dns config: %v", err)
	}

	defer teardown()

	go func() {
		for {
			select {
			case <-notifier.Connect():
				log.Infof("Client gRPC connection established")
			case <-notifier.Disconnect():
				log.Infof("Client gRPC connection shut down")
				teardown()
			}
		}
	}()

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
		sig := <-interrupt
		log.Infof("Received %s, shutting down gracefully.", sig)
		grpcServer.Stop()
	}()

	go func() {
		<-programContext.Done()
		grpcServer.Stop()
	}()

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("failed to start gRPC server: %v", err)
	}
	log.Infof("gRPC server shut down.")
}
