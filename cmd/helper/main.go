package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nais/device/pkg/helper"
	"github.com/nais/device/pkg/helper/config"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/unixsocket"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
)

var cfg = helper.Config{}

func init() {
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&cfg.Interface, "interface", "utun69", "interface name")

	flag.Parse()
}

func main() {
	programContext, cancel := context.WithCancel(context.Background())
	// for windows service control, noop on other unix
	err := helper.StartService(programContext, cancel)
	if err != nil {
		log.Fatalf("Starting windows service: %v", err)
	}

	osConfigurator := helper.New(cfg)

	logger.SetupLogger(cfg.LogLevel, config.LogDir, "helper.log")

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
	grpcServer := grpc.NewServer(grpc.StatsHandler(notifier))

	dhs := &helper.DeviceHelperServer{
		Config:         cfg,
		OSConfigurator: osConfigurator,
	}
	pb.RegisterDeviceHelperServer(grpcServer, dhs)

	teardown := func() {
		ctx, cancel := context.WithTimeout(programContext, time.Second*2)
		defer cancel()
		_, err := dhs.Teardown(ctx, &pb.TeardownRequest{})
		if err != nil {
			log.Warn(err)
		}
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
