package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"

	deviceagent "github.com/nais/device/pkg/device-agent"
	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/device-agent/filesystem"
	"github.com/nais/device/pkg/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/notify"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/unixsocket"
	"github.com/nais/device/pkg/version"
)

var (
	cfg = config.DefaultConfig()
)

func init() {
	flag.StringVar(&cfg.APIServer, "apiserver", cfg.APIServer, "base url to apiserver")
	flag.StringVar(&cfg.APIServerGRPCAddress, "apiserver-grpc-address", cfg.APIServerGRPCAddress, "grpc address to apiserver")
	flag.StringVar(&cfg.BootstrapAPI, "bootstrap-api", cfg.BootstrapAPI, "url to bootstrap API")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "path to agent config directory")
	flag.StringVar(&cfg.Interface, "interface", cfg.Interface, "name of tunnel interface")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")
	flag.StringVar(&cfg.GrpcAddress, "grpc-address", cfg.GrpcAddress, "unix socket for gRPC server")
	flag.StringVar(&cfg.DeviceAgentHelperAddress, "device-agent-helper-address", cfg.DeviceAgentHelperAddress, "device-agent-helper unix socket")
}

func main() {
	flag.Parse()
	cfg.SetDefaults()

	logDir := filepath.Join(cfg.ConfigDir, "logs")
	logger.SetupLogger(cfg.LogLevel, logDir, "agent.log")

	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Level:   sentry.LevelInfo,
		Message: "main",
		Type:    "type",
		Data: map[string]interface{}{
			"key": "value",
		},
		Category: "category",
	})

	environment := "production"
	if version.Version == "unknown" {
		environment = "development"
	}
	err := sentry.Init(sentry.ClientOptions{
		AttachStacktrace: true,
		Debug:            true,
		Release:          version.Version,
		Dist:             config.Platform,
		Environment:      environment,
		// fixme: consider hiding this somewhere
		Dsn: "https://f71422489ffe4731a59c5268159f1c09@sentry.gc.nav.no/93",
	})
	if err != nil {
		log.Fatalf("BUG: Setup sentry sdk: %s", err)
	}
	defer sentry.Flush(2 * time.Second)

	sentry.CaptureMessage("device-agent starting up")

	cfg.PopulateAgentConfiguration()

	log.Infof("naisdevice-agent %s starting up", version.Version)
	log.Infof("configuration: %+v", cfg)

	err = startDeviceAgent(&cfg)
	if err != nil {
		notify.Errorf(err.Error())
		log.Errorf("naisdevice-agent terminated with error.")
		os.Exit(1)
	}

	log.Infof("naisdevice-agent shutting down.")
}

func startDeviceAgent(cfg *config.Config) error {
	if err := filesystem.EnsurePrerequisites(cfg); err != nil {
		return fmt.Errorf("missing prerequisites: %s", err)
	}

	rc, err := runtimeconfig.New(cfg)
	if err != nil {
		log.Errorf("instantiate runtime config: %v", err)
		return fmt.Errorf("unable to start naisdevice-agent, check logs for details")
	}

	log.Infof("naisdevice-helper connection on unix socket %s", cfg.DeviceAgentHelperAddress)
	connection, err := grpc.Dial(
		"unix:"+cfg.DeviceAgentHelperAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("connect to naisdevice-helper: %v", err)
	}

	client := pb.NewDeviceHelperClient(connection)
	defer connection.Close()

	listener, err := unixsocket.ListenWithFileMode(cfg.GrpcAddress, 0666)
	if err != nil {
		return err
	}
	log.Infof("accepting network connections on unix socket %s", cfg.GrpcAddress)

	grpcServer := grpc.NewServer()
	das := deviceagent.NewServer(client, cfg, rc)
	pb.RegisterDeviceAgentServer(grpcServer, das)

	go func() {
		das.EventLoop()
		grpcServer.Stop()
	}()

	err = grpcServer.Serve(listener)
	if err != nil {
		return fmt.Errorf("failed to start gRPC server: %v", err)
	}

	log.Infof("gRPC server shut down.")

	return nil
}
