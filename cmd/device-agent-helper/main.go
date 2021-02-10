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

type MockService struct{}

func (service *MockService) ControlChannel() <-chan ControlEvent {
	return make(chan ControlEvent, 1)
}

type ControlEvent int

type Controllable interface {
	ControlChannel() <-chan ControlEvent
}

var (
	cfg = device_helper.Config{}
)

const (
	Stop ControlEvent = iota
	Pause
	Continue
)

func init() {
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&cfg.ConfigPath, "config-dir", "", "path to naisdevice config dir (required)")
	flag.StringVar(&cfg.Interface, "interface", "utun69", "interface name")
	flag.StringVar(&cfg.GrpcAddress, "grpc-address", "", "interface name")

	flag.Parse()

	cfg.WireGuardConfigPath = filepath.Join(cfg.ConfigPath, cfg.Interface+".conf")
}

func main() {
	if len(cfg.ConfigPath) == 0 {
		fmt.Println("--config-dir is required")
		os.Exit(1)
	}

	if len(cfg.GrpcAddress) == 0 {
		cfg.GrpcAddress = filepath.Join(cfg.ConfigPath, "helper.sock")
	}

	logger.SetupDeviceLogger(cfg.LogLevel, logger.DeviceAgentHelperLogFilePath())

	log.Infof("Starting device-agent-helper with config:\n%+v", cfg)

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		device_helper.TeardownInterface(ctx, cfg.Interface)
		cancel()
	}()

	if err := device_helper.Prerequisites(); err != nil {
		log.Fatalf("Checking prerequisites: %v", err)
	}

	// Deprecated service, new one is installed via msi intaller
	device_helper.UninstallService()

	listener, err := unixsocket.ListenWithFileMode(cfg.GrpcAddress, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("accepting network connections on unix socket %s", cfg.GrpcAddress)

	grpcServer := grpc.NewServer()
	dhs := &device_helper.DeviceHelperServer{
		Config: cfg,
	}
	pb.RegisterDeviceHelperServer(grpcServer, dhs)

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
