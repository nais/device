package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/nais/device/pkg/logger"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = Config{}
)

type Config struct {
	Interface           string
	BinaryDir           string
	WireGuardBinary     string
	WireGuardGoBinary   string
	WireGuardConfigPath string
	LogLevel            string
	DeviceIP            string
}

func init() {
	flag.StringVar(&cfg.Interface, "interface", "", "name of tunnel interface")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&cfg.WireGuardConfigPath, "wireguard-config-path", "", "path to the WireGuard-config the helper will actuate")
	flag.StringVar(&cfg.WireGuardBinary, "wireguard-binary", cfg.WireGuardBinary, "path to WireGuard binary")
	platformFlags(&cfg)

	flag.Parse()

	logger.Setup(cfg.LogLevel, true)
}

// device-agent-helper is responsible for:
// - running the WireGuard process
// - configuring the network tunnel interface
// - synchronizing WireGuard with the provided config
// - setting up the required routes
func main() {
	log.Infof("Starting device-agent-helper with config:\n%+v", cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := prerequisites(); err != nil {
		log.Fatalf("Checking prerequisites: %v", err)
	}

	if err := setupInterface(ctx, cfg); err != nil {
		log.Fatalf("Setting up interface: %v", err)
	}
	defer teardownInterface(ctx, cfg)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	lastSync := time.Time{}
	for {
		select {
		case <-interrupt:
			log.Info("Received interrupt, shutting down gracefully.")
			return

		case <-time.After(10 * time.Second):
			info, err := os.Stat(cfg.WireGuardConfigPath)
			if err != nil {
				log.Errorf("checking WireGuard config stats: %v", err)
			}

			if info.ModTime().After(lastSync) {
				err = syncConf(cfg, ctx)
				if err != nil {
					log.Errorf("Syncing WireGuard config: %v", err)
				} else {
					lastSync = info.ModTime()
				}
			}
		}
	}
}

func ParseConfig(wireGuardConfig string) ([]string, error) {
	re := regexp.MustCompile(`AllowedIPs = (.+)`)
	allAllowedIPs := re.FindAllStringSubmatch(wireGuardConfig, -1)

	var cidrs []string

	for _, allowedIPs := range allAllowedIPs {
		cidrs = append(cidrs, strings.Split(allowedIPs[1], ",")...)
	}

	return cidrs, nil
}

func filesExist(files ...string) error {
	for _, file := range files {
		if err := FileMustExist(file); err != nil {
			return err
		}
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

func runCommands(ctx context.Context, commands [][]string) error {
	for _, s := range commands {
		cmd := exec.CommandContext(ctx, s[0], s[1:]...)

		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("running %v: %w: %v", cmd, err, string(out))
		} else {
			log.Debugf("cmd: %v: %v\n", cmd, string(out))
		}
	}
	return nil
}
