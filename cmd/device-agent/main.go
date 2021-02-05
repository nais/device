package main

import (
	"fmt"
	"github.com/gen2brain/beeep"
	"github.com/nais/device/device-agent/filesystem"
	"github.com/nais/device/device-agent/runtimeconfig"
	device_agent "github.com/nais/device/pkg/device-agent"
	pb "github.com/nais/device/pkg/protobuf"
	"google.golang.org/grpc"
	"net"

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
	flag.BoolVar(&cfg.AutoConnect, "connect", false, "auto connect")
	flag.Parse()
	cfg.SetDefaults()
	logger.SetupDeviceLogger(cfg.LogLevel, cfg.LogFilePath)
}

func main() {
	log.Infof("Starting device-agent with config:\n%+v", cfg)
	log.Infof("Version: %s, Revision: %s", version.Version, version.Revision)
	startDeviceAgent()
}

func startDeviceAgent() {
	if err := filesystem.EnsurePrerequisites(&cfg); err != nil {
		notify(fmt.Sprintf("Missing prerequisites: %s", err))
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", 0))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	log.Infof("listening on port %d", port)
	cfg.GrpcPort = port

	rc, err := runtimeconfig.New(cfg)
	if err != nil {
		log.Errorf("Runtime config: %v", err)
		notify("Unable to start naisdevice, check logs for details")
		return
	}

	var opts []grpc.ServerOption

	stateChange := make(chan device_agent.ProgramState, 64)

	grpcServer := grpc.NewServer(opts...)
	das := device_agent.NewServer(stateChange)
	pb.RegisterDeviceAgentServer(grpcServer, das)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to start grpc webserver: %v", err)
		}
	}()

	das.EventLoop(rc)
}

func notify(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	err := beeep.Notify("NAIS device", message, "../Resources/nais-logo-red.png")
	log.Infof("sending message to notification centre: %s", message)
	if err != nil {
		log.Errorf("failed sending message due to error: %s", err)
	}
}
