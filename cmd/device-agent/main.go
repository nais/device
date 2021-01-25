package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/getlantern/systray"
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
	systray.Run(onReady, onExit)
}

func DeleteConfigFile(path string) error {
	err := os.Remove(path)
	if err != nil && err != os.ErrNotExist {
		return err
	}
	log.Debugf("Removed WireGuard configuration file at %s", path)
	lastConfigurationFile = ""
	return nil
}

func WriteConfigFile(path string, r io.Reader) error {
	cfg, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, cfg, 0600)
	if err != nil {
		return fmt.Errorf("writing WireGuard config to disk: %w", err)
	}

	log.Debugf("Wrote WireGuard config to disk")
	return nil
}
