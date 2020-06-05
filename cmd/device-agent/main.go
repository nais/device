package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/config"
	"github.com/nais/device/device-agent/filesystem"
	"github.com/nais/device/device-agent/runtimeconfig"
	"github.com/nais/device/device-agent/wireguard"
	"github.com/nais/device/pkg/logger"
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
	flag.StringVar(&cfg.BinaryDir, "binary-dir", cfg.BinaryDir, "path to binary directory")
	flag.StringVar(&cfg.Interface, "interface", cfg.Interface, "name of tunnel interface")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")
	flag.Parse()

	logger.Setup(cfg.LogLevel, true)
}

func main() {
	log.Infof("Starting device-agent with config:\n%+v", cfg)
	cfg.SetDefaults()

	if err := filesystem.EnsurePrerequisites(&cfg); err != nil {
		log.Fatalf("Verifying prerequisites: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rc, err := runtimeconfig.New(cfg, ctx)
	if err != nil {
		log.Fatalf("Initializing runtime config: %v", err)
	}

	baseConfig := wireguard.GenerateBaseConfig(rc.BootstrapConfig, rc.PrivateKey)

	if err := ioutil.WriteFile(cfg.WireGuardConfigPath, []byte(baseConfig), 0600); err != nil {
		log.Fatalf("Writing base WireGuard config to disk: %v", err)
	}

	fmt.Println("Starting device-agent-helper, you might be prompted for password")

	if err := runHelper(rc, ctx); err != nil {
		log.Errorf("Running helper: %v", err)
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-interrupt:
			log.Info("Received interrupt, shutting down gracefully.")
			return

		case <-time.After(5 * time.Second):
			if err := SyncConfig(baseConfig, rc); err != nil {
				log.Errorf("Unable to synchronize config with apiserver: %v", err)
			}
		}
	}
}

func SyncConfig(baseConfig string, rc *runtimeconfig.RuntimeConfig) error {
	gateways, err := apiserver.GetGateways(rc.Client, rc.Config.APIServer, rc.Serial)

	if err != nil {
		return fmt.Errorf("unable to get gateway config: %w", err)
	}

	wireGuardPeers := wireguard.GenerateWireGuardPeers(gateways)

	if err := ioutil.WriteFile(rc.Config.WireGuardConfigPath, []byte(baseConfig+wireGuardPeers), 0600); err != nil {
		return fmt.Errorf("writing WireGuard config to disk: %w", err)
	}

	log.Debugf("Wrote WireGuard config to disk")
	return nil
}
