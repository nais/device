package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/device/pkg/bootstrap"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/nais/device/pkg/logger"
	"github.com/rjeczalik/notify"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = Config{}
)

const (
	TunnelNetworkPrefix = "10.255.24"
)

type Config struct {
	Interface           string
	WireGuardBinary     string
	WireGuardGoBinary   string
	WireGuardConfigPath string
	ConfigPath          string
	LogLevel            string
	BootstrapConfig     *bootstrap.Config
	BootstrapConfigPath string
}

func init() {
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&cfg.ConfigPath, "config-dir", "", "path to naisdevice config dir")
	flag.StringVar(&cfg.Interface, "interface", "utun69", "interface name")
	platformFlags(&cfg)

	flag.Parse()

	cfg.WireGuardGoBinary = filepath.Join("/", "Applications", "naisdevice.app", "Contents", "MacOS", "wireguard-go")
	cfg.WireGuardBinary = filepath.Join("/", "Applications", "naisdevice.app", "Contents", "MacOS", "wg")
	cfg.WireGuardConfigPath = filepath.Join(cfg.ConfigPath, cfg.Interface+".conf")
	cfg.BootstrapConfigPath = filepath.Join(cfg.ConfigPath, "bootstrapconfig.json")

	logger.Setup(cfg.LogLevel)
}

// device-agent-helper is responsible for:
// - running the WireGuard process
// - configuring the network tunnel interface
// - synchronizing WireGuard with the provided config
// - setting up the required routes
func main() {
	log.Infof("Starting device-agent-helper with config:\n%+v", cfg)

	var err error
	cfg.BootstrapConfig, err = parseBootstrapConfig(cfg)
	if err != nil {
		log.Fatal("parsing bootstrap config", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := prerequisites(); err != nil {
		log.Fatalf("Checking prerequisites: %v", err)
	}

	defer teardownInterface(ctx, cfg)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	configdir := path.Dir(cfg.WireGuardConfigPath)
	notifyEvents := make(chan notify.EventInfo, 100)
	err = notify.Watch(configdir, notifyEvents, notify.Remove, notify.Write)
	if err != nil {
		log.Fatalf("Monitoring WireGuard configuration file: %v", err)
	}

	ensureUp := func() {
		log.Info("WireGuard configuration updated")
		if err := setupInterface(ctx, cfg); err != nil {
			log.Errorf("Setting up interface: %v", err)
			return
		}
		err = syncConf(cfg, ctx)
		if err != nil {
			log.Errorf("Syncing WireGuard config: %v", err)
		}
	}

	ensureDown := func() {
		log.Info("WireGuard configuration deleted; tearing down interface")
		teardownInterface(ctx, cfg)
	}

	if RegularFileExists(cfg.WireGuardConfigPath) == nil {
		ensureUp()
	} else {
		ensureDown()
	}

	for {
		select {
		case <-interrupt:
			log.Info("Received interrupt, shutting down gracefully.")
			return

		case ev := <-notifyEvents:
			log.Infof("%#v", ev)
			if ev.Path() != cfg.WireGuardConfigPath {
				continue
			}
			switch ev.Event() {
			case notify.Remove:
				ensureDown()
			case notify.Write:
				ensureUp()
			}
		}
	}
}

func parseBootstrapConfig(cfg Config) (*bootstrap.Config, error) {
	b, err := ioutil.ReadFile(cfg.BootstrapConfigPath)
	if err != nil {
		return nil, fmt.Errorf("reading bootstrapconfig.json: %w", err)
	}

	var bootstrapConfig bootstrap.Config
	err = json.Unmarshal(b, &bootstrapConfig)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling bootstrapconfig.json: %w", err)
	}

	return &bootstrapConfig, nil
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
		if err := RegularFileExists(file); err != nil {
			return err
		}
	}

	return nil
}

func RegularFileExists(filepath string) error {
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

		time.Sleep(100 * time.Millisecond) // avoid serializable race conditions with kernel
	}
	return nil
}
