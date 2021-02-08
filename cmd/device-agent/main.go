package main

import (
	"fmt"
	"net"

	"github.com/gen2brain/beeep"
	"github.com/nais/device/device-agent/filesystem"
	"github.com/nais/device/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/device-agent"
	pb "github.com/nais/device/pkg/protobuf"
	"google.golang.org/grpc"

	"github.com/nais/device/device-agent/config"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = config.DefaultConfig()
)

func init() {
	flag.StringVar(&cfg.APIServer, "apiserver", cfg.APIServer, "base url to apiserver")
	flag.StringVar(&cfg.BootstrapAPI, "bootstrap-api", cfg.BootstrapAPI, "url to bootstrap API")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "path to agent config directory")
	flag.StringVar(&cfg.Interface, "interface", cfg.Interface, "name of tunnel interface")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")
	flag.StringVar(&cfg.GrpcAddress, "grpc-address", cfg.GrpcAddress, "unix socket for gRPC server")
	flag.BoolVar(&cfg.AutoConnect, "connect", false, "auto connect")
	flag.Parse()
	cfg.SetDefaults()
	logger.SetupDeviceLogger(cfg.LogLevel, cfg.LogFilePath)
}

func main() {
	log.Infof("Starting device-agent with config:\n%+v", cfg)
	log.Infof("Version: %s, Revision: %s", version.Version, version.Revision)
	startDeviceAgent()
	log.Infof("device-agent shutting down.")
}

func startDeviceAgent() {
	if err := filesystem.EnsurePrerequisites(&cfg); err != nil {
		notify(fmt.Sprintf("Missing prerequisites: %s", err))
	}

	rc, err := runtimeconfig.New(cfg)
	if err != nil {
		log.Errorf("Runtime config: %v", err)
		notify("Unable to start naisdevice, check logs for details")
		return
	}

	// fixme: socket is not cleaned up when process is killed with SIGKILL,
	// fixme: and requires manual removal.
	listener, err := net.Listen("unix", cfg.GrpcAddress)
	if err != nil {
		log.Fatalf("failed to listen on unix socket %s: %v", cfg.GrpcAddress, err)
	}
	log.Infof("accepting network connections on unix socket %s", cfg.GrpcAddress)

	var opts []grpc.ServerOption

	stateChange := make(chan pb.AgentState, 64)

	grpcServer := grpc.NewServer(opts...)
	das := device_agent.NewServer(stateChange)
	pb.RegisterDeviceAgentServer(grpcServer, das)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to start grpc webserver: %v", err)
		}
	}()

	das.EventLoop(rc)

	grpcServer.Stop()
	listener.Close()
}

func notify(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	err := beeep.Notify("NAIS device", message, "../Resources/nais-logo-red.png")
	log.Infof("sending message to notification centre: %s", message)
	if err != nil {
		log.Errorf("failed sending message due to error: %s", err)
	}
}
