package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	deviceagent "github.com/nais/device/internal/device-agent"
	"github.com/nais/device/internal/device-agent/acceptableuse"
	"github.com/nais/device/internal/device-agent/agenthttp"
	"github.com/nais/device/internal/device-agent/auth"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/filesystem"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/logger"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/program"
	"github.com/nais/device/internal/unixsocket"
	"github.com/nais/device/internal/version"
	"github.com/nais/device/pkg/pb"
)

const (
	healthCheckInterval  = 20 * time.Second // how often to healthcheck gateways
	versionCheckInterval = 1 * time.Hour    // how often to check for a new version of naisdevice
)

func main() {
	cfg, err := config.DefaultConfig()
	if err != nil {
		logrus.StandardLogger().WithError(err).Error("unable to read default configuration")
		os.Exit(1)
	}

	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "path to agent config directory")
	flag.StringVar(&cfg.Interface, "interface", cfg.Interface, "name of tunnel interface")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")
	flag.StringVar(&cfg.GrpcAddress, "grpc-address", cfg.GrpcAddress, "unix socket for gRPC server")
	flag.BoolVar(&cfg.NoHelper, "no-helper", false, "stop the agent from configuring the machine through the helper")
	flag.StringVar(&cfg.GoogleAuthServerAddress, "google-auth-server-address", cfg.GoogleAuthServerAddress, "Google auth-server address")
	flag.BoolVar(&cfg.LocalAPIServer, "local-apiserver", false, "Connect to a local apiserver on 127.0.0.1:8099 using mock authentication")
	flag.StringVar(&cfg.CustomEnrollURL, "custom-enroll-url", "", "Connect to a custom enroller")
	flag.Parse()

	cfg.SetDefaults()

	logDir := filepath.Join(cfg.ConfigDir, logger.LogDir)
	log := logger.SetupLogger(cfg.LogLevel, logDir, logger.Agent).WithField("component", "main")
	defer logger.CapturePanic(log)

	ctx, cancel := program.MainContext(time.Second)
	defer cancel()

	cfg.PopulateAgentConfiguration(log)

	log.WithFields(version.LogFields).WithField("cfg", *cfg).Info("starting naisdevice-agent")

	notifier := notify.New(log)
	err = run(ctx, log, cfg, notifier)
	if err != nil {
		notifier.ShowError(err)
		log.WithError(err).Error("naisdevice-agent terminated")
		os.Exit(1)
	}

	log.Info("naisdevice-agent shutting down")
}

func run(ctx context.Context, log *logrus.Entry, cfg *config.Config, notifier notify.Notifier) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	otelCancel, err := otel.SetupOTelSDK(ctx, "naisdevice-agent", log)
	if err != nil {
		return fmt.Errorf("setup OTel SDK: %s", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := otelCancel(ctx); err != nil {
			log.WithError(err).Error("shutdown OTel SDK")
		}
		cancel()
	}()

	if err := filesystem.EnsurePrerequisites(cfg); err != nil {
		return fmt.Errorf("missing prerequisites: %s", err)
	}

	rc, err := runtimeconfig.New(log.WithField("component", "runtimeconfig"), cfg)
	if err != nil {
		log.WithError(err).Error("instantiate runtime config")
		return fmt.Errorf("unable to start naisdevice-agent, check logs for details")
	}

	if cfg.AgentConfiguration.ILoveNinetiesBoybands {
		err := rc.PopulateTenants(ctx)
		if err != nil {
			return fmt.Errorf("populate tenants from bucket: %w", err)
		}
	}

	log.WithField("helper_address", cfg.DeviceHelperAddress).Info("naisdevice-helper connection")

	var client pb.DeviceHelperClient
	if cfg.LocalAPIServer || cfg.NoHelper {
		client = pb.NewMockHelperClient(log)
	} else {
		connection, err := grpc.NewClient(
			"unix:"+cfg.DeviceHelperAddress,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithIdleTimeout(10*time.Hour),
			grpc.WithStatsHandler(otel.NewGRPCClientHandler(pb.DeviceHelper_Ping_FullMethodName)),
		)
		if err != nil {
			return fmt.Errorf("connect to naisdevice-helper: %v", err)
		}
		client = pb.NewDeviceHelperClient(connection)
		defer connection.Close()
	}

	go func() {
		var helperCheckErrors []error
		for ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(healthCheckInterval):
				err = helperHealthCheck(ctx, client)
				if err != nil {
					helperCheckErrors = append(helperCheckErrors, err)
					if len(helperCheckErrors) < 3 {
						break
					}

					log.WithField("errors", helperCheckErrors).Error("unable to communicate with helper, agent shutting down")
					notifier.Errorf("unable to communicate with helper, agent shutting down")
					cancel()
				}
				// healthcheck successful, reset errors
				helperCheckErrors = nil
			}
		}
	}()

	authHandler := auth.New(cfg.GoogleAuthServerAddress)

	listener, err := unixsocket.ListenWithFileMode(cfg.GrpcAddress, 0o666)
	if err != nil {
		return err
	}
	log.WithField("grpc_address", cfg.GrpcAddress).Info("accepting network connections on unix socket")

	statusChannel := make(chan *pb.AgentStatus, 32)
	stateMachine := deviceagent.NewStateMachine(ctx, rc, *cfg, notifier, client, statusChannel, authHandler, log.WithField("component", "statemachine"))

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	acceptableUseHandler := acceptableuse.New(rc, log.WithField("component", "acceptable-use"))
	acceptableUseHandler.Register(agenthttp.HandleFunc)

	das := deviceagent.NewServer(ctx, log.WithField("component", "device-agent-server"), cfg, rc, notifier, stateMachine.SendEvent, acceptableUseHandler)
	pb.RegisterDeviceAgentServer(grpcServer, das)

	newVersionChannel := make(chan bool, 1)
	go versionChecker(ctx, newVersionChannel, notifier, log, rc)

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

	httpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("creating listener: %w", err)
	}

	log.WithField("addr", httpListener.Addr().String()).Info("local HTTP server")
	go func() {
		if err := agenthttp.Serve(httpListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Error("HTTP server")
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

	log.Info("gRPC server shut down")

	return nil
}

func versionChecker(ctx context.Context, newVersionChannel chan<- bool, notifier notify.Notifier, log logrus.FieldLogger, rc runtimeconfig.RuntimeConfig) {
	versionCheckTimer := time.NewTimer(versionCheckInterval)
	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return
		case <-versionCheckTimer.C:
			newVersionAvailable, err := checkNewVersionAvailable(ctx)
			if err != nil {
				log.WithError(err).Info("check for new version")
				break
			}

			newVersionChannel <- newVersionAvailable
			if newVersionAvailable {
				url := "https://docs.nais.io/how-to-guides/naisdevice/update"
				domain := rc.GetDomainFromToken()
				if domain != "default" { // if parsing fail we get default
					url = fmt.Sprintf("https://docs.%s.cloud.nais.io/how-to-guides/naisdevice/update", domain)
				}
				notifier.Infof("New version of device agent available: " + url)
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

	if _, err := client.Ping(helperHealthCheckCtx, &pb.PingRequest{}); err != nil {
		return err
	}
	return nil
}

func checkNewVersionAvailable(ctx context.Context) (bool, error) {
	type response struct {
		Tag string `json:"tag_name"`
	}

	ctx, span := otel.Start(ctx, "CheckNewVersionAvailable")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/nais/device/releases/latest", nil)
	if err != nil {
		span.RecordError(err)
		return false, err
	}
	resp, err := otelhttp.DefaultClient.Do(req)
	if err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("retrieve current release version: %s", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status code from GitHub API when checking for new version: %d", resp.StatusCode)
		span.RecordError(err)
		return false, err
	}

	res := &response{}
	if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
		span.RecordError(err)
		return false, fmt.Errorf("unmarshal response: %s", err)
	}

	if res.Tag != "" && version.Version != res.Tag {
		return true, nil
	}

	return false, nil
}
