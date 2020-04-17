package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = DefaultConfig()
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	flag.StringVar(&cfg.Name, "name", cfg.Name, "gateway name")
	flag.StringVar(&cfg.TunnelIP, "tunnel-ip", cfg.TunnelIP, "gateway tunnel ip")
	flag.StringVar(&cfg.APIServerURL, "api-server-url", cfg.APIServerURL, "api server URL")
	flag.StringVar(&cfg.APIServerPublicKey, "api-server-public-key", cfg.APIServerPublicKey, "api server public key")
	flag.StringVar(&cfg.APIServerWireGuardEndpoint, "api-server-wireguard-endpoint", cfg.APIServerWireGuardEndpoint, "api server WireGuard endpoint")

	flag.Parse()

	cfg.WireGuardConfigPath = path.Join(cfg.ConfigDir, "wg0.conf")
	cfg.PrivateKeyPath = path.Join(cfg.ConfigDir, "private.key")
}

// Gateway agent ensures desired configuration as defined by the apiserver
// is synchronized and enforced by the local wireguard process on the gateway.
//
// Prerequisites:
// - controlplane tunnel is set up/apiserver is reachable at `Config.APIServerURL`
//
// Prereqs for MVP (at least):
//
// - wireguard keypair is generated and provided as `Config.{Public,Private}Key`
// - gateway is registered
// - tunnel ip is configured on wireguard interface for dataplane (see below)
//
//# ip link add dev wg0 type wireguard
//# ip addr add <tunnelip> wg0
//# ip link set wg0 up
type Device struct {
	Serial    string     `json:"serial"`
	PSK       string     `json:"psk"`
	LastCheck *time.Time `json:"lastCheck"`
	Healthy   *bool      `json:"isHealthy"`
	PublicKey string     `json:"publicKey"`
	IP        string     `json:"ip"`
}

func main() {
	log.Info("starting gateway-agent")
	log.Infof("with config:\n%+v", cfg)
	if err := setupInterface(cfg.TunnelIP); err != nil {
		log.Fatalf("setting up interface: %v", err)
	}

	privateKey, err := readPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		log.Fatalf("reading private key: %s", err)
	}

	baseConfig := GenerateBaseConfig(cfg, privateKey)
	if err := actuateWireGuardConfig(baseConfig, cfg.WireGuardConfigPath); err != nil {
		log.Fatalf("actuating base config: %v", err)
	}

	for range time.NewTicker(10 * time.Second).C {
		log.Infof("getting config")
		devices, err := getDevices(cfg.APIServerURL, cfg.Name)
		if err != nil {
			log.Error(err)
			// inc metric
			continue
		}

		log.Debugf("%+v\n", devices)

		peerConfig := GenerateWireGuardPeers(devices)
		if err := actuateWireGuardConfig(baseConfig+peerConfig, cfg.WireGuardConfigPath); err != nil {
			log.Errorf("actuating WireGuard config: %v", err)
		}
	}
}

func readPrivateKey(privateKeyPath string) (string, error) {
	b, err := ioutil.ReadFile(privateKeyPath)
	return string(b), err
}

func getDevices(apiServerURL, name string) ([]Device, error) {
	gatewayConfigURL := fmt.Sprintf("%s/gateways/%s", apiServerURL, name)
	resp, err := http.Get(gatewayConfigURL)
	if err != nil {
		return nil, fmt.Errorf("getting peer config from apiserver: %w", err)
	}

	defer resp.Body.Close()

	var devices []Device
	err = json.NewDecoder(resp.Body).Decode(&devices)

	if err != nil {
		return nil, fmt.Errorf("unmarshal json from apiserver: %w", err)
	}

	return devices, nil
}

type Config struct {
	APIServerURL               string
	Name                       string
	TunnelIP                   string
	ConfigDir                  string
	WireGuardConfigPath        string
	APIServerPublicKey         string
	APIServerWireGuardEndpoint string
	PrivateKeyPath             string
	APIServerTunnelIP          string
}

func DefaultConfig() Config {
	return Config{
		APIServerURL:      "http://apiserver.device.nais.io",
		APIServerTunnelIP: "10.255.240.1",
		ConfigDir:         "/usr/local/etc/nais-device",
	}
}

func setupInterface(tunnelIP string) error {
	if err := exec.Command("ip", "link", "del", "wg0").Run(); err != nil {
		log.Infof("pre-deleting WireGuard interface (ok if this fails): %v", err)
	}

	run := func(commands [][]string) error {
		for _, s := range commands {
			cmd := exec.Command(s[0], s[1:]...)

			if out, err := cmd.Output(); err != nil {
				return fmt.Errorf("running %v: %w", cmd, err)
			} else {
				fmt.Printf("%v: %v\n", cmd, string(out))
			}
		}
		return nil
	}

	commands := [][]string{
		{"ip", "link", "add", "dev", "wg0", "type", "wireguard"},
		{"ip", "link", "set", "wg0", "mtu", "1380"},
		{"ip", "address", "add", "dev", "wg0", tunnelIP},
		{"ip", "link", "set", "wg0", "up"},
	}

	return run(commands)
}

func GenerateBaseConfig(cfg Config, privateKey string) string {
	template := `[Interface]
PrivateKey = %s

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`

	return fmt.Sprintf(template, privateKey, cfg.APIServerPublicKey, cfg.APIServerTunnelIP, cfg.APIServerWireGuardEndpoint)
}

func GenerateWireGuardPeers(devices []Device) string {
	peerTemplate := `[Peer]
PublicKey = %s
AllowedIPs = %s
`
	var peers string

	for _, device := range devices {
		peers += fmt.Sprintf(peerTemplate, device.PublicKey, device.IP)
	}

	return peers
}

// actuateWireGuardConfig runs syncconfig with the provided WireGuard config
func actuateWireGuardConfig(wireGuardConfig, wireGuardConfigPath string) error {
	if err := ioutil.WriteFile(wireGuardConfigPath, []byte(wireGuardConfig), 0600); err != nil {
		return fmt.Errorf("writing WireGuard config to disk: %w", err)
	}

	cmd := exec.Command("wg", "syncconf", "wg0", wireGuardConfigPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running syncconf: %w", err)
	}

	log.Debugf("Actuated WireGuard config: %v", wireGuardConfigPath)

	return nil
}
