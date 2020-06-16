package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/getlantern/systray"
	"github.com/nais/device/device-agent/auth"
	"github.com/nais/device/device-agent/runtimeconfig"
	"github.com/nais/device/device-agent/wireguard"
	"github.com/nais/device/pkg/logger"
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
	log.Infof("Starting device-agent with config:\n%+v", cfg)

	systray.Run(onReady, onExit)

}

func ensureValidSessionInfo(apiserverURL, platform, serial string, ctx context.Context) (*auth.SessionInfo, error) {
	authURL, err := getAuthURL(apiserverURL, ctx)

	if err != nil {
		return nil, fmt.Errorf("getting Azure auth URL from apiserver: %v", err)
	}

	return auth.RunFlow(ctx, authURL, apiserverURL, platform, serial)
}

func getAuthURL(apiserverURL string, ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiserverURL+"/authurl", nil)
	if err != nil {
		return "", fmt.Errorf("creating request to get Azure auth URL: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("getting Azure auth URL from apiserver: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unable to get Azure auth URL from apiserver (%v), http status: %v", apiserverURL, resp.Status)
	}

	authURL, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read response body: %v", err)
	}
	return string(authURL), nil
}

func TruncateConfigFile(path string) error {
	f, err := ConfigFileDescriptor(path)
	if err == nil {
		log.Debugf("Truncated WireGuard configuration file at %s", path)
		f.Close()
	}
	return err
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
