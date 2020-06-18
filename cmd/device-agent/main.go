package main

import (
	"fmt"
	"io"
	"os"

	"github.com/getlantern/systray"
	"github.com/nais/device/device-agent/runtimeconfig"
	"github.com/nais/device/device-agent/wireguard"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
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
	cfg.SetDefaults()
	log.Infof("Starting device-agent with config:\n%+v", cfg)
	log.Infof("Version: %s, Revision: %s", version.Version, version.Revision)
	systray.Run(onReady, onExit)
}

func DeleteConfigFile(path string) error {
	err := os.Remove(path)
	if err != nil {
		return err
	}
	log.Debugf("Removed WireGuard configuration file at %s", path)
	return nil
}

func ConfigFileDescriptor(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
}

func WriteConfigFile(path string, rc runtimeconfig.RuntimeConfig) error {
	f, err := ConfigFileDescriptor(path)
	if err != nil {
		return err
	}
	defer f.Close()
	err = WriteConfig(f, rc)
	if err != nil {
		return fmt.Errorf("writing WireGuard config to disk: %w", err)
	}

	log.Debugf("Wrote WireGuard config to disk")
	return nil
}

func WriteConfig(w io.Writer, rc runtimeconfig.RuntimeConfig) error {
	baseConfig := wireguard.GenerateBaseConfig(rc.BootstrapConfig, rc.PrivateKey)
	_, err := w.Write([]byte(baseConfig))
	if err != nil {
		return err
	}

	wireGuardPeers := rc.Gateways.MarshalIni()
	_, err = w.Write(wireGuardPeers)
	if err != nil {
		return err
	}

	return err
}
