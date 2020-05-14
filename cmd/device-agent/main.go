package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/azure"
	"github.com/nais/device/device-agent/config"
	"github.com/nais/device/device-agent/serial"
	"github.com/nais/device/device-agent/wireguard"
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
	setPlatformDefaults(&cfg)
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

// device-agent is responsible for enabling the end-user to connect to it's permitted gateways.
// To be able to connect, a series of prerequisites must be in place. These will be helped/ensured by device-agent.
//
// A information exchange between end-user and NAIS device administrator/slackbot:
// If BootstrapTokenPath is not present, user will be prompted to enroll using a generated token, and the agent will exit.
// When device-agent detects a valid bootstrap token, it will generate a WireGuard config file called wg0.conf placed in `cfg.ConfigDir`
// This file will initially only contain the Interface definition and the APIServer peer.
//
// It will run the device-agent-helper with params....
//
// loop:
// Fetch device config from APIServer and configure generate and write WireGuard config to disk
// loop:
// Monitor all connections, if one starts failing, re-fetch config and reset timer
func main() {
	log.Infof("Starting device-agent with config:\n%+v", cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := platformPrerequisites(cfg)
	if err != nil {
		log.Errorf("Verifying platform prerequisites: %v", err)
		return
	}

	if err := filesExist(cfg.WireGuardBinary); err != nil {
		log.Errorf("Verifying if file exists: %v", err)
		return
	}

	if err := ensureDirectories(cfg.ConfigDir); err != nil {
		log.Errorf("Ensuring directory exists: %v", err)
		return
	}

	if err := ensureKey(cfg.PrivateKeyPath); err != nil {
		log.Errorf("Ensuring private key exists: %v", err)
		return
	}

	privateKeyEncoded, err := ioutil.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		log.Errorf("Reading private key: %v", err)
		return
	}

	privateKey, err := wireguard.Base64toKey(privateKeyEncoded)
	if err != nil {
		log.Errorf("Decoding private key: %v", err)
		return
	}

	deviceSerial, err := serial.GetDeviceSerial()
	if err != nil {
		log.Errorf("Getting device serial: %v", err)
		return
	}

	if err := filesExist(cfg.BootstrapTokenPath); err != nil {
		enrollmentToken, err := apiserver.GenerateEnrollmentToken(deviceSerial, cfg.Platform, wireguard.KeyToBase64(wireguard.WGPubKey(privateKey)))
		if err != nil {
			log.Errorf("Generating enrollment token: %v", err)
			return
		}

		fmt.Printf("\n---\nno bootstrap token present. Send 'naisdevice' your enrollment token on slack by copying and pasting this:\n/msg @naisdevice enroll %v\n\n", enrollmentToken)
		return
	}

	bootstrapToken, err := ioutil.ReadFile(cfg.BootstrapTokenPath)
	if err != nil {
		log.Errorf("Reading bootstrap token: %v", err)
		return
	}

	cfg.BootstrapConfig, err = apiserver.ParseBootstrapToken(string(bootstrapToken))
	if err != nil {
		log.Errorf("Parsing bootstrap config: %v", err)
		return
	}

	baseConfig := GenerateBaseConfig(cfg.BootstrapConfig, privateKey)

	if err := ioutil.WriteFile(cfg.WireGuardConfigPath, []byte(baseConfig), 0600); err != nil {
		log.Errorf("Writing WireGuard config to disk: %v", err)
		return
	}

	log.Debugf("Wrote base WireGuard config to disk")

	fmt.Println("Starting device-agent-helper, you might be prompted for password")

	if err = runHelper(ctx, cfg); err != nil {
		log.Errorf("Running helper: %v", err)
		return
	}

	token, err := azure.RunAuthFlow(ctx, cfg.OAuth2Config)
	if err != nil {
		log.Errorf("Unable to get token for user: %v", err)
		return
	}

	client := cfg.OAuth2Config.Client(ctx, token)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-interrupt:
			log.Info("Received interrupt, shutting down gracefully.")
			return

		case <-time.After(15 * time.Second):
			gateways, err := apiserver.GetGateways(client, cfg.APIServer, deviceSerial)
			if err != nil {
				log.Errorf("Unable to get gateway config: %v", err)
			}

			wireGuardPeers := wireguard.GenerateWireGuardPeers(gateways)

			if err := ioutil.WriteFile(cfg.WireGuardConfigPath, []byte(baseConfig+wireGuardPeers), 0600); err != nil {
				log.Errorf("Writing WireGuard config to disk: %v", err)
				return
			}

			log.Debugf("Wrote WireGuard config to disk")
		}
	}
}

func filesExist(files ...string) error {
	for _, file := range files {
		if err := FileMustExist(file); err != nil {
			return err
		}
	}

	return nil
}

func ensureDirectories(dirs ...string) error {
	for _, dir := range dirs {
		if err := ensureDirectory(dir); err != nil {
			return err
		}
	}

	return nil
}

func ensureDirectory(dir string) error {
	info, err := os.Stat(dir)

	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0700)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%v is a file", dir)
	}

	return nil
}

func ensureKey(keyPath string) error {
	if err := FileMustExist(keyPath); os.IsNotExist(err) {
		return ioutil.WriteFile(keyPath, wireguard.KeyToBase64(wireguard.WgGenKey()), 0600)
	} else if err != nil {
		return err
	}

	return nil
}

func FileMustExist(filepath string) error {
	info, err := os.Stat(filepath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%v is a directory", filepath)
	}

	return nil
}

func adminCommandContext(ctx context.Context, command string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "sudo", append([]string{command}, arg...)...)
}
