package main

import (
	"fmt"

	"github.com/gen2brain/beeep"
	"github.com/nais/device/device-agent/config"
	"github.com/nais/device/device-agent/filesystem"
	"github.com/nais/device/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/device-agent"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/unixsocket"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
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
	flag.StringVar(&cfg.DeviceAgentHelperAddress, "device-agent-helper-address", cfg.DeviceAgentHelperAddress, "device-agent-helper unix socket")
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

	log.Infof("device-agent-helper connection on unix socket %s", cfg.DeviceAgentHelperAddress)
	connection, err := grpc.Dial(
		"unix:"+cfg.DeviceAgentHelperAddress,
		grpc.WithInsecure(),

	)
	if err != nil {
		log.Fatalf("unable to connect to device-agent-helper grpc server: %v", err)
	}

	client := pb.NewDeviceHelperClient(connection)
	defer connection.Close()

	listener, err := unixsocket.ListenWithFileMode(cfg.GrpcAddress, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("accepting network connections on unix socket %s", cfg.GrpcAddress)

	grpcServer := grpc.NewServer()
	das := device_agent.NewServer(client)
	pb.RegisterDeviceAgentServer(grpcServer, das)

	go func() {
		das.EventLoop(rc)
		grpcServer.Stop()
	}()

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("failed to start gRPC server: %v", err)
	}
	log.Infof("gRPC server shut down.")
}

func notify(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	err := beeep.Notify("NAIS device", message, "../Resources/nais-logo-red.png")
	log.Infof("sending message to notification centre: %s", message)
	if err != nil {
		log.Errorf("failed sending message due to error: %s", err)
	}
}
