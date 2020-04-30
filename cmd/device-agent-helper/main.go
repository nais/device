package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

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
	TunnelIP            string
}

func init() {
	flag.StringVar(&cfg.Interface, "interface", "", "name of tunnel interface")
	flag.StringVar(&cfg.TunnelIP, "tunnel-ip", "", "device tunnel ip")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&cfg.WireGuardConfigPath, "wireguard-config-path", "", "path to the WireGuard-config the helper will actuate")
	flag.StringVar(&cfg.WireGuardBinary, "wireguard-binary", "", "path to WireGuard binary")
	flag.StringVar(&cfg.WireGuardGoBinary, "wireguard-go-binary", "", "path to WireGuard-go binary")

	flag.Parse()

	log.SetFormatter(&log.JSONFormatter{})
	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)
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

	if err := filesExist(cfg.WireGuardBinary, cfg.WireGuardGoBinary); err != nil {
		log.Fatalf("Verifying if file exists: %v", err)
	}

	if err := setupInterface(ctx, cfg); err != nil {
		log.Fatalf("Setting up interface: %v", err)
	}

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-interrupt:
			log.Info("Received interrupt, shutting down gracefully.")
			return

		case <-time.After(10 * time.Second):
			cmd := exec.CommandContext(ctx, cfg.WireGuardBinary, "syncconf", cfg.Interface, cfg.WireGuardConfigPath)
			if b, err := cmd.CombinedOutput(); err != nil {
				log.Errorf("Running syncconf: %v: %v", err, string(b))
			}

			configFileBytes, err := ioutil.ReadFile(cfg.WireGuardConfigPath)
			if err != nil {
				log.Errorf("Reading file: %v", err)
			}

			cidrs, err := ParseConfig(string(configFileBytes))
			if err != nil {
				log.Errorf("Parsing WireGuard config: %v", err)
			}

			if err := setupRoutes(ctx, cidrs, cfg.Interface); err != nil {
				log.Errorf("Setting up routes: %v", err)
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

func setupRoutes(ctx context.Context, cidrs []string, interfaceName string) error {
	for _, cidr := range cidrs {
		cmd := exec.CommandContext(ctx, "route", "-q", "-n", "add", "-inet", cidr, "-interface", interfaceName)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Errorf("%v: %v", cmd, string(output))
			return fmt.Errorf("executing %v: %w", cmd, err)
		}
		log.Debugf("%v: %v", cmd, string(output))
	}
	return nil
}

func setupInterface(ctx context.Context, cfg Config) error {
	run := func(commands [][]string) error {
		for _, s := range commands {
			cmd := exec.CommandContext(ctx, s[0], s[1:]...)

			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("running %v: %w: %v", cmd, err, string(out))
			} else {
				fmt.Printf("%v: %v\n", cmd, string(out))
			}
		}
		return nil
	}

	ip := cfg.TunnelIP
	commands := [][]string{
		{cfg.WireGuardGoBinary, cfg.Interface},
		{"ifconfig", cfg.Interface, "inet", ip + "/21", ip, "add"},
		{"ifconfig", cfg.Interface, "mtu", "1360"},
		{"ifconfig", cfg.Interface, "up"},
		{"route", "-q", "-n", "add", "-inet", ip + "/21", "-interface", cfg.Interface},
	}

	return run(commands)
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
