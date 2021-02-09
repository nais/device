package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/nais/device/pkg/device-helper"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/pb"
	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := device_helper.Prerequisites(); err != nil {
		log.Fatalf("Checking prerequisites: %v", err)
	}

	// Deprecated service, new one is installed via msi intaller
	device_helper.UninstallService()

	defer device_helper.TeardownInterface(ctx, cfg.Interface)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// fixme: socket is not cleaned up when process is killed with SIGKILL,
	// fixme: and requires manual removal.
	listener, err := net.Listen("unix", cfg.GrpcAddress)
	if err != nil {
		log.Fatalf("failed to listen on unix socket %s: %v", cfg.GrpcAddress, err)
	}
	log.Infof("accepting network connections on unix socket %s", cfg.GrpcAddress)

	grpcServer := grpc.NewServer()
	dhs := &device_helper.DeviceHelperServer{}
	pb.RegisterDeviceHelperServer(grpcServer, dhs)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to start grpc webserver: %v", err)
		}
	}()

	sig := <-interrupt
	log.Infof("Received %s, shutting down gracefully.", sig)
}
