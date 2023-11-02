package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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

func handleSignals(log *logrus.Entry, cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
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
	cfg, err := config.DefaultConfig()
	if err != nil {
		logrus.StandardLogger().Errorf("unable to read default configuration: %v", err)
		os.Exit(1)
	}

	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "path to agent config directory")
	flag.StringVar(&cfg.Interface, "interface", cfg.Interface, "name of tunnel interface")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")
	flag.StringVar(&cfg.GrpcAddress, "grpc-address", cfg.GrpcAddress, "unix socket for gRPC server")
	flag.StringVar(&cfg.DeviceAgentHelperAddress, "device-agent-helper-address", cfg.DeviceAgentHelperAddress, "device-agent-helper unix socket")
	flag.StringVar(&cfg.GoogleAuthServerAddress, "google-auth-server-address", cfg.GoogleAuthServerAddress, "Google auth-server address")
	flag.Parse()

	cfg.SetDefaults()

	logDir := filepath.Join(cfg.ConfigDir, logger.LogDir)
	log := logger.SetupLogger(cfg.LogLevel, logDir, logger.Agent).WithField("component", "main")

	programContext, programCancel := context.WithCancel(context.Background())
	handleSignals(log, programCancel)

	cfg.PopulateAgentConfiguration(log)

	log.Infof("naisdevice-agent %s starting up", version.Version)
	log.Infof("configuration: %+v", cfg)

	notifier := notify.New(log)
	err = run(programContext, log, cfg, notifier)
	if err != nil {
		notifier.Errorf(err.Error())
		log.Errorf("naisdevice-agent terminated with error.")
		os.Exit(1)
	}

	log.Infof("naisdevice-agent shutting down.")
}

func run(ctx context.Context, log *logrus.Entry, cfg *config.Config, notifier notify.Notifier) error {
	if err := filesystem.EnsurePrerequisites(cfg); err != nil {
		return fmt.Errorf("missing prerequisites: %s", err)
	}

	rc, err := runtimeconfig.New(log, cfg)
	if err != nil {
		log.Errorf("instantiate runtime config: %v", err)
		return fmt.Errorf("unable to start naisdevice-agent, check logs for details")
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
		grpc.WithTransportCredentials(insecure.NewCredentials()),
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

	grpcServer := grpc.NewServer()
	das := deviceagent.NewServer(log, client, cfg, rc, notifier)
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
