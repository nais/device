package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nais/device/apiserver/api"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = DefaultConfig()
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	flag.StringVar(&cfg.Apiserver, "apiserver", cfg.Apiserver, "hostname to apiserver")
	flag.StringVar(&cfg.Name, "name", cfg.Name, "hostname to apiserver")
	flag.StringVar(&cfg.PublicKey, "public-key", cfg.PublicKey, "path to wireguard public key")
	flag.StringVar(&cfg.PrivateKey, "private-key", cfg.PrivateKey, "path to wireguard private key")
	flag.StringVar(&cfg.TunnelInterfaceName, "interface", cfg.TunnelInterfaceName, "name of tunnel interface")
	flag.StringVar(&cfg.TunnelConfigDir, "tunnel-config-dir", cfg.TunnelConfigDir, "path to tunnel interface config directory")

	flag.Parse()
}

// Gateway agent ensures desired configuration as defined by the apiserver
// is synchronized and enforced by the local wireguard process on the gateway.
//
// Prerequisites:
// - controlplane tunnel is set up/apiserver is reachable at `Config.Apiserver`
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
func main() {
	log.Info("starting gateway-agent")
	log.Infof("with config:\n%+v", cfg)

	privateKey, err := readPrivateKey()

	if err != nil {
		log.Fatalf("reading private key: %s", err)
	}

	for range time.NewTicker(10 * time.Second).C {
		log.Infof("getting config")
		apiserver := fmt.Sprintf("%s/gateways/%s", cfg.Apiserver, cfg.Name)
		peers, err := getPeers(apiserver)
		if err != nil {
			log.Error(err)
			// inc metric
			continue
		}

		fmt.Printf("%+v\n", peers)

		if err := configureWireguard(peers, privateKey); err != nil {
			log.Error(err)
			// inc metric
		}
	}
}

func readPrivateKey() (string, error) {
	b, err := ioutil.ReadFile(cfg.PrivateKey)
	return string(b), err
}

func configureWireguard(peers []api.Peer, privateKey string) error {
	wgConfigContent := generateWGConfig(peers, privateKey)
	fmt.Println(string(wgConfigContent))
	wgConfigFilePath := filepath.Join(cfg.TunnelConfigDir, cfg.TunnelInterfaceName+".conf")
	if err := ioutil.WriteFile(wgConfigFilePath, wgConfigContent, 0600); err != nil {
		return fmt.Errorf("writing wireguard config to disk: %s", err)
	}

	cmd := exec.Command("/usr/bin/wg", "syncconf", "wgdata", "/etc/wireguard/wgdata.conf")
	if stdout, err := cmd.Output(); err != nil {
		return fmt.Errorf("executing %s: %s: %s", cmd, err, string(stdout))
	}

	return nil
}

func generateWGConfig(peers []api.Peer, privateKey string) []byte {
	wgConfig := "[Interface]\n"
	wgConfig += fmt.Sprintf("PrivateKey = %s\n", strings.Trim(privateKey, "\n"))
	wgConfig += "ListenPort = 51820\n"
	for _, peer := range peers {
		wgConfig += "[Peer]\n"
		wgConfig += fmt.Sprintf("PublicKey = %s\n", peer.PublicKey)
		wgConfig += fmt.Sprintf("AllowedIPs = %s\n\n", peer.IP)
	}

	return []byte(wgConfig)
}

func getPeers(apiserverURL string) (peers []api.Peer, err error) {
	resp, err := http.Get(apiserverURL)
	if err != nil {
		return nil, fmt.Errorf("getting peer config from apiserver: %s", err)
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&peers)

	if err != nil {
		return nil, fmt.Errorf("unmarshal json from apiserver: %s", err)
	}

	return
}

type Config struct {
	Apiserver           string
	Name                string
	PublicKey           string
	PrivateKey          string
	TunnelIP            string
	TunnelConfigDir     string
	TunnelInterfaceName string
}

func DefaultConfig() Config {
	return Config{
		Apiserver:           "http://apiserver.device.nais.io",
		PublicKey:           "/etc/wireguard/public.key",
		PrivateKey:          "/etc/wireguard/private.key",
		TunnelInterfaceName: "wgdata",
		TunnelConfigDir:     "/etc/wireguard/",
	}
}
