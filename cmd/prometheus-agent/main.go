package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/nais/device/pkg/ioconvenience"
	"github.com/nais/device/pkg/logger"
)

const GatewayFetchInterval = 5 * time.Minute

var (
	cfg                 = DefaultConfig()
	failedConfigFetches = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "failed_config_fetches",
		Help:      "count of failed config fetches",
		Namespace: "naisdevice",
		Subsystem: "prometheus_agent",
	})
	lastSuccessfulConfigFetch = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "last_successful_config_fetch",
		Help:      "time since last successful config fetch",
		Namespace: "naisdevice",
		Subsystem: "prometheus_agent",
	})
)

func init() {
	logger.Setup(cfg.LogLevel)
	flag.StringVar(&cfg.TunnelIP, "tunnel-ip", cfg.TunnelIP, "prometheus tunnel ip")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "agent configuration directory")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.APIServerURL, "api-server-url", cfg.APIServerURL, "api server URL")
	flag.StringVar(&cfg.APIServerPublicKey, "api-server-public-key", cfg.APIServerPublicKey, "api server public key")
	flag.StringVar(&cfg.APIServerWireGuardEndpoint, "api-server-wireguard-endpoint", cfg.APIServerWireGuardEndpoint, "api server WireGuard endpoint")
	flag.BoolVar(&cfg.DevMode, "development-mode", cfg.DevMode, "development mode avoids setting up interface and configuring WireGuard")
	flag.StringVar(&cfg.APIServerUsername, "apiserver-username", cfg.APIServerUsername, "apiserver username")
	flag.StringVar(&cfg.APIServerPassword, "apiserver-password", cfg.APIServerPassword, "apiserver password")

	flag.Parse()

	cfg.WireGuardConfigPath = path.Join(cfg.ConfigDir, "wg0.conf")
	cfg.PrivateKeyPath = path.Join(cfg.ConfigDir, "private.key")
	prometheus.MustRegister(failedConfigFetches)
	prometheus.MustRegister(lastSuccessfulConfigFetch)
}

type Gateway struct {
	PublicKey string `json:"publicKey"`
	IP        string `json:"ip"`
	Endpoint  string `json:"endpoint"`
}

func main() {
	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	log.Info("starting prometheus-agent")
	log.Infof("with config:\n%+v", cfg)

	if !cfg.DevMode {
		if err := setupInterface(cfg.TunnelIP); err != nil {
			log.Fatalf("setting up interface: %v", err)
		}
	} else {
		log.Infof("Skipping interface setup")
	}

	privateKey, err := readPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		log.Fatalf("reading private key: %s", err)
	}

	baseConfig := GenerateBaseConfig(cfg, privateKey)
	if err := actuateWireGuardConfig(baseConfig, cfg.WireGuardConfigPath, cfg.DevMode); err != nil {
		log.Fatalf("actuating base config: %v", err)
	}

	timer := time.NewTimer(1 * time.Millisecond)
	for range timer.C {
		timer.Reset(GatewayFetchInterval)
		log.Infof("getting config")
		gateways, err := getGateways(cfg)
		if err != nil {
			log.Error(err)
			failedConfigFetches.Inc()
			continue
		}

		nodeTargetsFile, err := os.Create("/etc/prometheus/node-targets.json")
		if err != nil {
			log.Fatalf("Unable to open file: %v", err)
		}

		if err := WritePrometheusTargets(gateways, 9100, nodeTargetsFile); err != nil {
			log.Fatalf("Unable to write prometheus config: %v", err)
		}
		_ = nodeTargetsFile.Close()

		gatewayTargetsFile, err := os.Create("/etc/prometheus/gateway-targets.json")
		if err != nil {
			log.Fatalf("Unable to open file: %v", err)
		}

		if err := WritePrometheusTargets(gateways, 3000, gatewayTargetsFile); err != nil {
			log.Fatalf("Unable to write prometheus config: %v", err)
		}
		_ = gatewayTargetsFile.Close()

		lastSuccessfulConfigFetch.SetToCurrentTime()

		log.Debugf("%+v\n", gateways)

		peerConfig := GenerateWireGuardPeers(gateways)
		if err := actuateWireGuardConfig(baseConfig+peerConfig, cfg.WireGuardConfigPath, cfg.DevMode); err != nil {
			log.Errorf("actuating WireGuard config: %v", err)
		}
	}
}

func WritePrometheusTargets(gateways []Gateway, port int, writer io.Writer) error {
	var targets []string
	for _, gw := range gateways {
		targets = append(targets, fmt.Sprintf("%v:%v", gw.IP, port))
	}

	return json.NewEncoder(writer).Encode([]PrometheusSDConfig{{Targets: targets}})
}

type PrometheusSDConfig struct {
	Targets []string `json:"targets"`
}

func readPrivateKey(privateKeyPath string) (string, error) {
	b, err := ioutil.ReadFile(privateKeyPath)
	return string(b), err
}

func getGateways(config Config) ([]Gateway, error) {
	prometheusConfigURL := fmt.Sprintf("%s/gateways", config.APIServerURL)
	req, err := http.NewRequest(http.MethodGet, prometheusConfigURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	req.SetBasicAuth(config.APIServerUsername, config.APIServerPassword)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting gateways from apiserver: %w", err)
	}

	defer ioconvenience.CloseReader(resp.Body)

	var gateways []Gateway
	err = json.NewDecoder(resp.Body).Decode(&gateways)

	if err != nil {
		return nil, fmt.Errorf("unmarshal json from apiserver: %w", err)
	}

	return gateways, nil
}

type Config struct {
	APIServerURL               string
	APIServerUsername          string
	APIServerPassword          string
	TunnelIP                   string
	ConfigDir                  string
	WireGuardConfigPath        string
	APIServerPublicKey         string
	APIServerWireGuardEndpoint string
	PrivateKeyPath             string
	APIServerTunnelIP          string
	DevMode                    bool
	PrometheusAddr             string
	PrometheusPublicKey        string
	PrometheusTunnelIP         string
	LogLevel                   string
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

func setupInterface(tunnelIP string) error {
	if err := exec.Command("ip", "link", "del", "wg0").Run(); err != nil {
		log.Infof("pre-deleting WireGuard interface (ok if this fails): %v", err)
	}

	run := func(commands [][]string) error {
		for _, s := range commands {
			cmd := exec.Command(s[0], s[1:]...)

			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("running %v: %w: %v", cmd, err, string(out))
			} else {
				fmt.Printf("%v: %v\n", cmd, string(out))
			}
		}
		return nil
	}

	commands := [][]string{
		{"ip", "link", "add", "dev", "wg0", "type", "wireguard"},
		{"ip", "link", "set", "wg0", "mtu", "1360"},
		{"ip", "address", "add", "dev", "wg0", tunnelIP + "/21"},
		{"ip", "link", "set", "wg0", "up"},
	}

	return run(commands)
}

func GenerateBaseConfig(cfg Config, privateKey string) string {
	template := `[Interface]
PrivateKey = %s
ListenPort = 51820

[Peer] # apiserver
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`

	return fmt.Sprintf(template, privateKey, cfg.APIServerPublicKey, cfg.APIServerTunnelIP, cfg.APIServerWireGuardEndpoint)
}

func GenerateWireGuardPeers(gateways []Gateway) string {
	peerTemplate := `[Peer]
PublicKey = %s
AllowedIPs = %s
Endpoint = %s
`
	var peers string

	for _, gateway := range gateways {
		peers += fmt.Sprintf(peerTemplate, gateway.PublicKey, gateway.IP, gateway.Endpoint)
	}

	return peers
}

// actuateWireGuardConfig runs syncconf with the provided WireGuard config
func actuateWireGuardConfig(wireGuardConfig, wireGuardConfigPath string, devMode bool) error {
	if err := ioutil.WriteFile(wireGuardConfigPath, []byte(wireGuardConfig), 0600); err != nil {
		return fmt.Errorf("writing WireGuard config to disk: %w", err)
	}

	cmd := exec.Command("wg", "syncconf", "wg0", wireGuardConfigPath)

	if !devMode {
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("running syncconf: %w", err)
		}
	} else {
		log.Infof("DevMode: would run %v here", cmd)
	}

	log.Debugf("Actuated WireGuard config: %v", wireGuardConfigPath)

	return nil
}
