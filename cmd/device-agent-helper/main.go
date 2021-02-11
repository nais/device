package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nais/device/pkg/device-helper"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/unixsocket"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
)

var (
	cfg = device_helper.Config{}
)

func init() {
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&cfg.ConfigDir, "config-dir", "", "path to naisdevice config dir (required)")
	flag.StringVar(&cfg.Interface, "interface", "utun69", "interface name")
	flag.StringVar(&cfg.GrpcAddress, "grpc-address", "", "interface name")

	flag.Parse()

	cfg.WireGuardConfigPath = filepath.Join(cfg.ConfigDir, cfg.Interface+".conf")
}

func main() {
	if len(cfg.ConfigDir) == 0 {
		fmt.Println("--config-dir is required")
		os.Exit(1)
	}

	if len(cfg.GrpcAddress) == 0 {
		cfg.GrpcAddress = filepath.Join(cfg.ConfigDir, "helper.sock")
	}

	logger.SetupLogger(cfg.LogLevel, cfg.ConfigDir, "helper.log")

	log.Infof("Starting device-agent-helper with config:\n%+v", cfg)

	osConfigurator := device_helper.New(cfg)

	if err := osConfigurator.Prerequisites(); err != nil {
		log.Fatalf("Checking prerequisites: %v", err)
	}

	listener, err := unixsocket.ListenWithFileMode(cfg.GrpcAddress, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("accepting network connections on unix socket %s", cfg.GrpcAddress)

	notifier := pb.NewConnectionNotifier()
	grpcServer := grpc.NewServer(grpc.StatsHandler(notifier))

	dhs := &device_helper.DeviceHelperServer{
		Config:         cfg,
		OSConfigurator: osConfigurator,
	}
	pb.RegisterDeviceHelperServer(grpcServer, dhs)

	teardown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
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

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("failed to start gRPC server: %v", err)
	}
	log.Infof("gRPC server shut down.")
}
