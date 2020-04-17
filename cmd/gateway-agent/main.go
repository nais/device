package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = DefaultConfig()
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	flag.StringVar(&cfg.Apiserver, "apiserver", cfg.Apiserver, "hostname to apiserver")
	flag.StringVar(&cfg.Name, "name", cfg.Name, "gateway name")
	flag.StringVar(&cfg.PublicKey, "public-key", cfg.PublicKey, "path to wireguard public key")
	flag.StringVar(&cfg.PrivateKeyPath, "private-key", cfg.PrivateKeyPath, "path to wireguard private key")
	flag.StringVar(&cfg.Interface, "interface", cfg.Interface, "name of tunnel interface")
	flag.StringVar(&cfg.TunnelConfigDir, "tunnel-config-dir", cfg.TunnelConfigDir, "path to tunnel interface config directory")

	flag.Parse()

	cfg.WireGuardConfigPath = path.Join("/etc/wireguard/", fmt.Sprintf("%s.connf", cfg.Interface))
}

// Gateway agent ensures desired configuration as defined by the apiserver
// is synchronized and enforced by the local wireguard process on the gateway.
//
// Prerequisites:
// - controlplane tunnel is set up/apiserver is reachable at `Config.APIServer`
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

	privateKey, err := readPrivateKey(cfg.PrivateKeyPath)

	if err != nil {
		log.Fatalf("reading private key: %s", err)
	}

	for range time.NewTicker(10 * time.Second).C {
		log.Infof("getting config")
		apiserver := fmt.Sprintf("%s/gateways/%s", cfg.Apiserver, cfg.Name)
		devices, err := getDevices(apiserver)
		if err != nil {
			log.Error(err)
			// inc metric
			continue
		}

		fmt.Printf("%+v\n", devices)

		if err := configureWireguard(devices, cfg, privateKey); err != nil {
			log.Error(err)
			// inc metric
		}
	}
}

func readPrivateKey(privateKeyPath string) (string, error) {
	b, err := ioutil.ReadFile(privateKeyPath)
	return string(b), err
}

func configureWireguard(devices []Device, cfg Config, privateKey string) error {
	wgConfigContent := generateWGConfig(devices, privateKey)
	fmt.Println(string(wgConfigContent))
	wgConfigFilePath := filepath.Join(cfg.TunnelConfigDir, cfg.Interface+".conf")
	if err := ioutil.WriteFile(wgConfigFilePath, wgConfigContent, 0600); err != nil {
		return fmt.Errorf("writing wireguard config to disk: %w", err)
	}

	cmd := exec.Command("/usr/bin/wg", "syncconf", cfg.Interface, cfg.WireGuardConfigPath)
	if stdout, err := cmd.Output(); err != nil {
		return fmt.Errorf("executing %w: %v", err, string(stdout))
	}

	return nil
}

func generateWGConfig(devices []Device, privateKey string) []byte {
	interfaceTemplate := `[Interface]
PrivateKey = %s
ListenPort = 51820

`

	wgConfig := fmt.Sprintf(interfaceTemplate, strings.TrimSuffix(privateKey, "\n"))

	peerTemplate := `[Peer]
AllowedIPs = %s/32
PublicKey = %s
`

	for _, device := range devices {
		wgConfig += fmt.Sprintf(peerTemplate, device.IP, device.PublicKey)
	}

	return []byte(wgConfig)
}

func getDevices(apiServerURL string) ([]Device, error) {
	resp, err := http.Get(apiServerURL)
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
	Apiserver           string
	Name                string
	PublicKey           string
	PrivateKeyPath      string
	TunnelIP            string
	TunnelConfigDir     string
	Interface           string
	WireGuardConfigPath string
}

func DefaultConfig() Config {
	return Config{
		Apiserver:       "http://apiserver.device.nais.io",
		PublicKey:       "/etc/wireguard/public.key",
		PrivateKeyPath:  "/etc/wireguard/private.key",
		Interface:       "wg0",
		TunnelConfigDir: "/etc/wireguard/",
	}
}
