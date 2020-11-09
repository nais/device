package config

import (
	"fmt"
	"github.com/coreos/go-iptables/iptables"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os/exec"
	"strings"
)

type Config struct {
	APIServerURL               string
	Name                       string
	TunnelIP                   string
	ConfigDir                  string
	WireGuardConfigPath        string
	APIServerPublicKey         string
	APIServerWireGuardEndpoint string
	PrivateKeyPath             string
	PrivateKey                 string
	APIServerTunnelIP          string
	DevMode                    bool
	PrometheusAddr             string
	PrometheusPublicKey        string
	PrometheusTunnelIP         string
	APIServerPassword          string
	APIServerPasswordPath      string
	LogLevel                   string
	IPTables                   *iptables.IPTables
	DefaultInterface           string
	DefaultInterfaceIP         string
}

func DefaultConfig() Config {
	return Config{
		APIServerURL:      "http://10.255.240.1",
		APIServerTunnelIP: "10.255.240.1",
		ConfigDir:         "/usr/local/etc/nais-device",
		PrometheusAddr:    ":3000",
		LogLevel:          "info",
	}
}

func (c *Config) InitLocalConfig() error {
	var err error
	c.PrivateKey, err = readFileToString(c.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("reading private key: %s", err)
	}
	c.APIServerPassword, err = readFileToString(c.APIServerPasswordPath)
	if err != nil {
		return fmt.Errorf("reading API server password: %s", err)
	}
	if len(c.APIServerPassword) == 0 {
		return fmt.Errorf("API server password file empty: %s", c.APIServerPasswordPath)
	}

	if !c.DevMode {
		c.DefaultInterface, c.DefaultInterfaceIP, err = getDefaultInterfaceInfo()
		if err != nil {
			log.Fatalf("Getting default interface info: %v", err)
		}
	}

	return nil
}

func getDefaultInterfaceInfo() (string, string, error) {
	cmd := exec.Command("ip", "route", "get", "1.1.1.1")
	out, err := cmd.CombinedOutput()

	if err != nil {
		return "", "", fmt.Errorf("getting default gateway: %w", err)
	}

	return parseDefaultInterfaceOutput(out)
}

func parseDefaultInterfaceOutput(output []byte) (string, string, error) {
	lines := strings.Split(string(output), "\n")
	parts := strings.Split(lines[0], " ")
	if len(parts) != 9 {
		log.Errorf("wrong number of parts in output: '%v', output: '%v'", len(parts), string(output))
	}

	interfaceName := parts[4]
	if len(interfaceName) < 4 {
		return "", "", fmt.Errorf("weird interface name: '%v'", interfaceName)
	}

	interfaceIP := parts[6]

	if len(strings.Split(interfaceIP, ".")) != 4 {
		return "", "", fmt.Errorf("weird interface ip: '%v'", interfaceIP)
	}

	return interfaceName, interfaceIP, nil
}

func readFileToString(filePath string) (string, error) {
	b, err := ioutil.ReadFile(filePath)
	return string(b), err
}
