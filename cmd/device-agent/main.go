package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nais/device/device-agent/config"
	device_agent "github.com/nais/device/device-agent/device_agent"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = config.DefaultConfig()
)

func init() {
	flag.StringVar(&cfg.APIServer, "apiserver", cfg.APIServer, "base url to apiserver")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "path to agent config directory")
	flag.StringVar(&cfg.BinaryDir, "binary-dir", cfg.BinaryDir, "path to binary directory")
	flag.StringVar(&cfg.Interface, "interface", cfg.Interface, "name of tunnel interface")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")

	flag.Parse()

	setPlatform(&cfg)
	cfg.PrivateKeyPath = filepath.Join(cfg.ConfigDir, "private.key")
	cfg.WireGuardConfigPath = filepath.Join(cfg.ConfigDir, "wg0.conf")
	cfg.BootstrapTokenPath = filepath.Join(cfg.ConfigDir, "bootstrap.token")

	log.SetFormatter(&log.JSONFormatter{})
	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)
}

func main() {
	log.Infof("Starting device-agent with config:\n%+v", cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d, err := device_agent.New(cfg)
	if err != nil {
		log.Errorf("Setting up device-agent: %v", err)
		return
	}

	fmt.Println("Starting device-agent-helper, you might be prompted for password")
	if err := d.RunHelper(ctx); err != nil {
		log.Errorf("Running helper: %v", err)
		return
	}

	if err := d.AuthUser(ctx); err != nil {
		log.Errorf("Authenticating user: %v", err)
		return
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-interrupt:
			log.Info("Received interrupt, shutting down gracefully.")
			return

		case <-time.After(15 * time.Second):
			if err := d.SyncConfig(); err != nil {
				log.Errorf("Unable to synchronize config with apiserver: %v", err)
			}
		}
	}
}
