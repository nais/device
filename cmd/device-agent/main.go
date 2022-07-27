package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/nais/device/pkg/outtune"
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

var cfg = config.DefaultConfig()

func init() {
	flag.StringVar(&cfg.APIServerGRPCAddress, "apiserver-grpc-address", cfg.APIServerGRPCAddress, "grpc address to apiserver")
	flag.StringVar(&cfg.BootstrapAPI, "bootstrap-api", cfg.BootstrapAPI, "url to bootstrap API")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "path to agent config directory")
	flag.StringVar(&cfg.Interface, "interface", cfg.Interface, "name of tunnel interface")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")
	flag.StringVar(&cfg.GrpcAddress, "grpc-address", cfg.GrpcAddress, "unix socket for gRPC server")
	flag.StringVar(&cfg.DeviceAgentHelperAddress, "device-agent-helper-address", cfg.DeviceAgentHelperAddress, "device-agent-helper unix socket")
	flag.StringVar(&cfg.GoogleAuthServerAddress, "google-auth-server-address", cfg.GoogleAuthServerAddress, "Google auth-server address")
}

func handleSignals(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Level:   sentry.LevelInfo,
			Message: "signal received",
			Type:    "debug",
			Data: map[string]any{
				"signal": sig,
			},
			Category: "eventloop",
		})
		log.Infof("signal handler: got signal '%s', canceling main context", sig)
		cancel()
		log.Info("signal handler: allowing 1s to stop gracefully...", sig)
		// normally cancelling the context will result in the program returning before the next lines are evaluated
		time.Sleep(1 * time.Second)
		log.Info("signal handler: force-exiting")
		os.Exit(0)
	}()
}

func main() {
	flag.Parse()
	cfg.SetDefaults()

	programContext, programCancel := context.WithCancel(context.Background())
	handleSignals(programCancel)

	logDir := filepath.Join(cfg.ConfigDir, "logs")
	logger.SetupLogger(cfg.LogLevel, logDir, "agent.log")

	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Level:   sentry.LevelInfo,
		Message: "main",
		Type:    "type",
		Data: map[string]any{
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

	err = startDeviceAgent(programContext, &cfg)
	if err != nil {
		notify.Errorf(err.Error())
		log.Errorf("naisdevice-agent terminated with error.")
		os.Exit(1)
	}

	log.Infof("naisdevice-agent shutting down.")
}

func startDeviceAgent(ctx context.Context, cfg *config.Config) error {
	if err := filesystem.EnsurePrerequisites(cfg); err != nil {
		return fmt.Errorf("missing prerequisites: %s", err)
	}

	rc, err := runtimeconfig.New(cfg)
	if err != nil {
		log.Errorf("instantiate runtime config: %v", err)
		return fmt.Errorf("unable to start naisdevice-agent, check logs for details")
	}

	rc.Tenants = []*pb.Tenant{
		{
			Name:           "NAV",
			AuthProvider:   pb.AuthProvider_Azure,
			OuttuneEnabled: true,
		},
	}

	if cfg.AgentConfiguration.ILoveNinetiesBoybands {
		err := rc.PopulateTenants(ctx)
		if err != nil {
			return fmt.Errorf("populate tenants from bucket: %w", err)
		}
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

	listener, err := unixsocket.ListenWithFileMode(cfg.GrpcAddress, 0o666)
	if err != nil {
		return err
	}
	log.Infof("accepting network connections on unix socket %s", cfg.GrpcAddress)

	ot := outtune.New(client)

	grpcServer := grpc.NewServer()
	das := deviceagent.NewServer(client, cfg, rc, ot)
	pb.RegisterDeviceAgentServer(grpcServer, das)

	go func() {
		das.EventLoop(ctx)
		grpcServer.Stop()
	}()

	err = grpcServer.Serve(listener)
	if err != nil {
		return fmt.Errorf("failed to start gRPC server: %v", err)
	}

	log.Infof("gRPC server shut down.")

	return nil
}
