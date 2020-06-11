package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/auth"
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

	// wait until helper has established tunnel to apiserver...

	if rc.SessionInfo, err = ensureValidSessionInfo(cfg.SessionInfoPath, cfg.APIServer, cfg.Platform, rc.Serial, ctx); err != nil {
		log.Errorf("Ensuring valid session key: %v", err)
		return
	}

	if err := writeToJSONFile(rc.SessionInfo, cfg.SessionInfoPath); err != nil {
		log.Errorf("Writing session info to disk: %v", err)
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
			if err := SyncConfig(baseConfig, rc, ctx); err != nil {
				log.Errorf("Unable to synchronize config with apiserver: %v", err)
			}
		}
	}
}

func writeToJSONFile(strct interface{}, path string) error {
	b, err := json.Marshal(&strct)
	if err != nil {
		return fmt.Errorf("marshaling struct into json: %w", err)
	}
	if err := ioutil.WriteFile(path, b, 0600); err != nil {
		return err
	}
	return nil
}

func fileExists(filepath string) bool {
	info, err := os.Stat(filepath)
	if err != nil || info.IsDir() {
		return false
	}
	return true
}

func ensureValidSessionInfo(sessionInfoFile, apiserverURL, platform, serial string, ctx context.Context) (*auth.SessionInfo, error) {
	if fileExists(sessionInfoFile) {
		b, err := ioutil.ReadFile(sessionInfoFile)
		if err != nil {
			return nil, fmt.Errorf("reading session key file: %v", err)
		}
		var si auth.SessionInfo

		_ = json.Unmarshal(b, &si) // Ignoring unmarshalling errors, fetch new session key instead

		const MinRemainingKeyValidity = 8 * time.Hour
		if time.Unix(si.Expiry, 0).After(time.Now().Add(MinRemainingKeyValidity)) {
			log.Debug("Using cached session key, as it is valid for more than 8 hours")
			return &si, nil
		}
	}

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

func SyncConfig(baseConfig string, rc *runtimeconfig.RuntimeConfig, ctx context.Context) error {
	gateways, err := apiserver.GetGateways(rc.SessionInfo.Key, rc.Config.APIServer, ctx)

	if ue, ok := err.(*apiserver.UnauthorizedError); ok {
		log.Errorf("Unauthorized access from apiserver: %v\nAssuming invalid session. Removing cached session and stopping process.", ue)

		if err := os.Remove(rc.Config.SessionInfoPath); err != nil {
			log.Errorf("Removing session info file: %v", err)
		}

		os.Exit(1)
	}

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
