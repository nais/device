package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	deviceagent "github.com/nais/device/internal/device-agent"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/filesystem"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/logger"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/unixsocket"
	"github.com/nais/device/internal/version"
)

const (
	healthCheckInterval  = 20 * time.Second // how often to healthcheck gateways
	versionCheckInterval = 1 * time.Hour    // how often to check for a new version of naisdevice
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := filesystem.EnsurePrerequisites(cfg); err != nil {
		return fmt.Errorf("missing prerequisites: %s", err)
	}

	otelCancel, err := otel.SetupOTelSDK(ctx)
	if err != nil {
		return fmt.Errorf("setup OTel SDK: %s", err)
	}
	defer func() {
		if err := otelCancel(ctx); err != nil {
			log.Errorf("shutdown OTel SDK: %s", err)
		}
	}()

	rc, err := runtimeconfig.New(log.WithField("component", "runtimeconfig"), cfg)
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
		grpc.WithIdleTimeout(10*time.Hour),
	)
	if err != nil {
		return fmt.Errorf("connect to naisdevice-helper: %v", err)
	}

	client := pb.NewDeviceHelperClient(connection)
	defer connection.Close()

	go func() {
		for ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(healthCheckInterval):
				err = helperHealthCheck(ctx, client)
				if err != nil {
					log.WithError(err).Errorf("Unable to communicate with helper. Shutting down")
					notifier.Errorf("Unable to communicate with helper. Shutting down.")
					cancel()
				}
			}
		}
	}()

	listener, err := unixsocket.ListenWithFileMode(cfg.GrpcAddress, 0o666)
	if err != nil {
		return err
	}
	log.Infof("accepting network connections on unix socket %s", cfg.GrpcAddress)

	statusChannel := make(chan *pb.AgentStatus, 32)
	stateMachine := deviceagent.NewStateMachine(ctx, rc, *cfg, notifier, client, statusChannel, log.WithField("component", "statemachine"))

	grpcServer := grpc.NewServer()
	das := deviceagent.NewServer(ctx, log.WithField("component", "device-agent-server"), cfg, rc, notifier, stateMachine.SendEvent)
	pb.RegisterDeviceAgentServer(grpcServer, das)

	newVersionChannel := make(chan bool, 1)
	go versionChecker(ctx, newVersionChannel, notifier, log)

	go func() {
		// This routine forwards status updates from the state machine to the device agent server
		newVersionAvailable := false
		for ctx.Err() == nil {
			select {
			case newVersionAvailable = <-newVersionChannel:
			case s := <-statusChannel:
				s.NewVersionAvailable = newVersionAvailable
				s.Tenants = rc.Tenants()
				das.UpdateAgentStatus(s)
			case <-ctx.Done():
			}
		}
	}()

	go func() {
		stateMachine.Run(ctx)
		// after state machine is done, stop the grpcServer
		grpcServer.Stop()
	}()

	err = grpcServer.Serve(listener)
	if err != nil {
		return fmt.Errorf("failed to start gRPC server: %v", err)
	}

	log.Infof("gRPC server shut down.")

	return nil
}

func versionChecker(ctx context.Context, newVersionChannel chan<- bool, notifier notify.Notifier, logger logrus.FieldLogger) {
	versionCheckTimer := time.NewTimer(versionCheckInterval)
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-versionCheckTimer.C:
			newVersionAvailable, err := checkNewVersionAvailable(ctx)
			if err != nil {
				logger.Infof("check for new version: %s", err)
				break
			}

			newVersionChannel <- newVersionAvailable
			if newVersionAvailable {
				notifier.Infof("New version of device agent available: https://doc.nais.io/how-to-guides/naisdevice/update")
				versionCheckTimer.Stop()
			} else {
				versionCheckTimer.Reset(versionCheckInterval)
			}
		}
	}
}

func helperHealthCheck(ctx context.Context, client pb.DeviceHelperClient) error {
	helperHealthCheckCtx, helperHealthCheckCancel := context.WithTimeout(ctx, 5*time.Second)
	defer helperHealthCheckCancel()

	if _, err := client.GetSerial(helperHealthCheckCtx, &pb.GetSerialRequest{}); err != nil {
		return err
	}
	return nil
}

func checkNewVersionAvailable(ctx context.Context) (bool, error) {
	type response struct {
		Tag string `json:"tag_name"`
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/nais/device/releases/latest", nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("retrieve current release version: %s", err)
	}

	defer resp.Body.Close()

	res := &response{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(res)
	if err != nil {
		return false, fmt.Errorf("unmarshal response: %s", err)
	}

	if version.Version != res.Tag {
		return true, nil
	}

	return false, nil
}
