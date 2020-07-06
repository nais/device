package main

import (
	"encoding/json"
	"fmt"
	"github.com/nais/device/pkg/logger"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/coreos/go-iptables/iptables"
	"github.com/nais/device/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg                       = DefaultConfig()
	failedConfigFetches       prometheus.Counter
	lastSuccessfulConfigFetch prometheus.Gauge
	registeredDevices         prometheus.Gauge
	connectedDevices          prometheus.Gauge
	currentVersion            prometheus.Counter
)

func init() {
	flag.StringVar(&cfg.Name, "name", cfg.Name, "gateway name")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "gateway-agent config directory")
	flag.StringVar(&cfg.TunnelIP, "tunnel-ip", cfg.TunnelIP, "gateway tunnel ip")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.APIServerURL, "api-server-url", cfg.APIServerURL, "api server URL")
	flag.StringVar(&cfg.APIServerPublicKey, "api-server-public-key", cfg.APIServerPublicKey, "api server public key")
	flag.StringVar(&cfg.APIServerPassword, "api-server-password", cfg.APIServerPassword, "api server password")
	flag.StringVar(&cfg.APIServerWireGuardEndpoint, "api-server-wireguard-endpoint", cfg.APIServerWireGuardEndpoint, "api server WireGuard endpoint")
	flag.StringVar(&cfg.PrometheusPublicKey, "prometheus-public-key", cfg.PrometheusPublicKey, "prometheus public key")
	flag.StringVar(&cfg.PrometheusTunnelIP, "prometheus-tunnel-ip", cfg.PrometheusTunnelIP, "prometheus tunnel ip")
	flag.BoolVar(&cfg.DevMode, "development-mode", cfg.DevMode, "development mode avoids setting up interface and configuring WireGuard")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "log level")

	flag.Parse()

	logger.Setup(cfg.LogLevel)
	cfg.WireGuardConfigPath = path.Join(cfg.ConfigDir, "wg0.conf")
	cfg.PrivateKeyPath = path.Join(cfg.ConfigDir, "private.key")
	initMetrics(cfg.Name)
	log.Infof("Version: %s, Revision: %s", version.Version, version.Revision)
}

func initMetrics(name string) {
	currentVersion = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "current_version",
		Help:        "current running version",
		Namespace:   "naisdevice",
		Subsystem:   "gateway_agent",
		ConstLabels: prometheus.Labels{"name": name, "version": version.Version},
	})
	failedConfigFetches = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "failed_config_fetches",
		Help:        "count of failed config fetches",
		Namespace:   "naisdevice",
		Subsystem:   "gateway_agent",
		ConstLabels: prometheus.Labels{"name": name, "version": version.Version},
	})
	lastSuccessfulConfigFetch = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "last_successful_config_fetch",
		Help:        "time since last successful config fetch",
		Namespace:   "naisdevice",
		Subsystem:   "gateway_agent",
		ConstLabels: prometheus.Labels{"name": name, "version": version.Version},
	})
	registeredDevices = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "number_of_registered_devices",
		Help:        "number of registered devices on a gateway",
		Namespace:   "naisdevice",
		Subsystem:   "gateway_agent",
		ConstLabels: prometheus.Labels{"name": name, "version": version.Version},
	})
	connectedDevices = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "number_of_connected_devices",
		Help:        "number of connected devices on a gateway",
		Namespace:   "naisdevice",
		Subsystem:   "gateway_agent",
		ConstLabels: prometheus.Labels{"name": name, "version": version.Version},
	})
	prometheus.MustRegister(failedConfigFetches)
	prometheus.MustRegister(lastSuccessfulConfigFetch)
	prometheus.MustRegister(registeredDevices)
	prometheus.MustRegister(connectedDevices)
	prometheus.MustRegister(currentVersion)
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
// # ip link add dev wg0 type wireguard
// # ip addr add <tunnelip> wg0
// # ip link set wg0 up
type GatewayConfig struct {
	Devices []Device `json:"devices"`
	Routes  []string `json:"routes"`
}

type Device struct {
	PSK       string `json:"psk"`
	PublicKey string `json:"publicKey"`
	IP        string `json:"ip"`
}

func main() {
	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	log.Info("starting gateway-agent")

	if !cfg.DevMode {
		if err := setupInterface(cfg.TunnelIP); err != nil {
			log.Fatalf("setting up interface: %v", err)
		}

		var err error
		cfg.IPTables, err = iptables.New()
		if err != nil {
			log.Fatalf("setting up iptables %v", err)
		}

		cfg.DefaultInterface, cfg.DefaultInterfaceIP, err = getDefaultInterfaceInfo()
		if err != nil {
			log.Fatalf("Getting default interface info: %v", err)
		}

		err = setupIptables(cfg)
		if err != nil {
			log.Fatalf("Setting up iptables defaults: %v", err)
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

	go checkForNewRelease(cfg)

	for range time.NewTicker(10 * time.Second).C {
		log.Infof("getting config")
		gatewayConfig, err := getGatewayConfig(cfg)
		if err != nil {
			log.Error(err)
			failedConfigFetches.Inc()
			continue
		}

		err = updateConnectedDevicesMetrics(cfg)
		if err != nil {
			log.Errorf("Unable to execute command: %v", err)
		}

		lastSuccessfulConfigFetch.SetToCurrentTime()

		log.Debugf("%+v\n", gatewayConfig)

		peerConfig := GenerateWireGuardPeers(gatewayConfig.Devices)
		if err := actuateWireGuardConfig(baseConfig+peerConfig, cfg.WireGuardConfigPath, cfg.DevMode); err != nil {
			log.Errorf("actuating WireGuard config: %v", err)
		}

		err = forwardRoutes(cfg, gatewayConfig.Routes)
		if err != nil {
			log.Errorf("forwarding routes: %v", err)
		}
	}
}

func readPrivateKey(privateKeyPath string) (string, error) {
	b, err := ioutil.ReadFile(privateKeyPath)
	return string(b), err
}

func getGatewayConfig(config Config) (*GatewayConfig, error) {
	gatewayConfigURL := fmt.Sprintf("%s/gatewayconfig", config.APIServerURL)
	req, err := http.NewRequest(http.MethodGet, gatewayConfigURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	req.SetBasicAuth(config.Name, config.APIServerPassword)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting peer config from apiserver: %w", err)
	}

	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading bytes, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching gatewayConfig from apiserver: %v %v %v", resp.StatusCode, resp.Status, string(b))
	}

	var gatewayConfig GatewayConfig
	err = json.Unmarshal(b, &gatewayConfig)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json from apiserver: bytes: %v, error: %w", string(b), err)
	}

	registeredDevices.Set(float64(len(gatewayConfig.Devices)))

	return &gatewayConfig, nil
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
	DevMode                    bool
	PrometheusAddr             string
	PrometheusPublicKey        string
	PrometheusTunnelIP         string
	APIServerPassword          string
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

[Peer] # prometheus
PublicKey = %s
AllowedIPs = %s/32

`

	return fmt.Sprintf(template, privateKey, cfg.APIServerPublicKey, cfg.APIServerTunnelIP, cfg.APIServerWireGuardEndpoint, cfg.PrometheusPublicKey, cfg.PrometheusTunnelIP)
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

func updateConnectedDevicesMetrics(cfg Config) error {
	if cfg.DevMode {
		connectedDevices.Set(1337)
		return nil
	}

	output, err := exec.Command("wg", "show", "wg0", "endpoints").Output()
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{1,5}`)
	matches := re.FindAll(output, -1)

	numConnectedDevices := float64(len(matches))
	connectedDevices.Set(numConnectedDevices)
	return nil
}

// actuateWireGuardConfig runs syncconfig with the provided WireGuard config
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

func checkForNewRelease(cfg Config) {
	type response struct {
		Tag string `json:"tag_name"`
	}

	for range time.NewTicker(120 * time.Second).C {
		log.Info("Checking release version on github")

		resp, err := http.Get("https://api.github.com/repos/nais/device/releases/latest")
		if err != nil {
			log.Errorf("Unable to retrieve current release version %s", err)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		res := response{}
		err = json.Unmarshal(body, &res)
		if err != nil {
			log.Errorf("unable to unmarshall response: %s", err)
			continue
		}

		if version.Version != res.Tag {
			log.Info("New version available. So long and thanks for all the fish.")
			if !cfg.DevMode {
				log.Info("jk, DevMode")
			} else {
				os.Exit(0)
			}
		}
	}
}

func setupIptables(cfg Config) error {
	err := cfg.IPTables.ChangePolicy("filter", "FORWARD", "DROP")
	if err != nil {
		return fmt.Errorf("setting FORWARD policy to DROP: %w", err)
	}

	// Allow ESTABLISHED,RELATED from wg0 to default interface
	err = cfg.IPTables.AppendUnique("filter", "FORWARD", "-i", "wg0", "-o", cfg.DefaultInterface, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default forward rule: %w", err)
	}

	// Allow ESTABLISHED,RELATED from default interface to wg0
	err = cfg.IPTables.AppendUnique("filter", "FORWARD", "-i", cfg.DefaultInterface, "-o", "wg0", "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default forward rule: %w", err)
	}

	return nil
}

func getDefaultInterfaceInfo() (string, string, error) {
	cmd := exec.Command("ip", "route", "get", "1.1.1.1")
	out, err := cmd.CombinedOutput()

	if err != nil {
		return "", "", fmt.Errorf("getting default gateway: %w", err)
	}

	return ParseDefaultInterfaceOutput(out)
}

func ParseDefaultInterfaceOutput(output []byte) (string, string, error) {
	lines := strings.Split(string(output), "\n")
	parts := strings.Split(lines[0], " ")
	if len(parts) != 9 {
		log.Errorf("wrong number of parts in output: '%v', output: '%v'", len(parts), string(output))
		//return "", "", fmt.Errorf("wrong number of parts in output: '%v', output: '%v'", len(parts), string(output))
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

func forwardRoutes(cfg Config, routes []string) error {
	var err error

	for _, ip := range routes {
		err = cfg.IPTables.AppendUnique("nat", "POSTROUTING", "-o", cfg.DefaultInterface, "-p", "tcp", "-d", ip, "-j", "SNAT", "--to-source", cfg.DefaultInterfaceIP)
		if err != nil {
			return fmt.Errorf("setting up snat: %w", err)
		}

		err = cfg.IPTables.AppendUnique("filter", "FORWARD", "-i", "wg0", "-o", cfg.DefaultInterface, "-p", "tcp", "--syn", "-d", ip, "-m", "conntrack", "--ctstate", "NEW", "-j", "ACCEPT")
		if err != nil {
			return fmt.Errorf("setting up forward: %w", err)
		}
	}

	return nil
}
