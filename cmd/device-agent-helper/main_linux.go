package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func prerequisites() error {
	if err := filesExist(cfg.WireGuardBinary, cfg.WireGuardGoBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}
	return nil
}

func platformFlags(cfg *Config) {
	flag.StringVar(&cfg.DeviceIP, "device-ip", "", "device tunnel ip")
	flag.StringVar(&cfg.WireGuardGoBinary, "wireguard-go-binary", "", "path to WireGuard-go binary")
}

func syncConf(cfg Config, ctx context.Context) error {
	cmd := exec.CommandContext(ctx, cfg.WireGuardBinary, "syncconf", cfg.Interface, cfg.WireGuardConfigPath)
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("running syncconf: %w: %v", err, string(b))
	}

	configFileBytes, err := ioutil.ReadFile(cfg.WireGuardConfigPath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	cidrs, err := ParseConfig(string(configFileBytes))
	if err != nil {
		return fmt.Errorf("parsing WireGuard config: %w", err)
	}

	if err := setupRoutes(ctx, cidrs, cfg.Interface); err != nil {
		return fmt.Errorf("setting up routes: %w", err)
	}

	return nil
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
	ip := cfg.DeviceIP
	commands := [][]string{
		{cfg.WireGuardGoBinary, cfg.Interface},
		{"ifconfig", cfg.Interface, "inet", ip + "/21", ip, "add"},
		{"ifconfig", cfg.Interface, "mtu", "1360"},
		{"ifconfig", cfg.Interface, "up"},
		{"route", "-q", "-n", "add", "-inet", ip + "/21", "-interface", cfg.Interface},
	}

	return runCommands(ctx, commands)
}

func teardownInterface(ctx context.Context, cfg Config) {
	return
}
