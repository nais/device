package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = DefaultConfig()
)

type Config struct {
	APIServer           string
	Interface           string
	ConfigDir           string
	BinaryDir           string
	BootstrapToken      string
	WireGuardPath       string
	WireGuardGoBinary   string
	PrivateKeyPath      string
	WireGuardConfigPath string
	BootstrapTokenPath  string
	BootstrapConfig     *BootstrapConfig
	LogLevel            string
}

type Gateway struct {
	PublicKey string `json:"publicKey"`
	Endpoint  string `json:"endpoint"`
	IP        string `json:"ip"`
}

func init() {
	flag.StringVar(&cfg.APIServer, "apiserver", cfg.APIServer, "base url to apiserver")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "path to agent config directory")
	flag.StringVar(&cfg.BinaryDir, "binary-dir", cfg.BinaryDir, "path to binary directory")
	flag.StringVar(&cfg.Interface, "interface", cfg.Interface, "name of tunnel interface")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")

	flag.Parse()

	cfg.WireGuardPath = filepath.Join(cfg.BinaryDir, "naisdevice-wg")
	cfg.WireGuardGoBinary = filepath.Join(cfg.BinaryDir, "naisdevice-wireguard-go")
	cfg.PrivateKeyPath = filepath.Join(cfg.ConfigDir, "private.key")
	cfg.WireGuardConfigPath = filepath.Join(cfg.ConfigDir, "wg0.conf")
	cfg.BootstrapTokenPath = filepath.Join(cfg.ConfigDir, "bootstrap.token")

	log.SetFormatter(&log.JSONFormatter{})
	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)
}

// device-agent is responsible for enabling the end-user to connect to it's permitted gateways.
// To be able to connect, a series of prerequisites must be in place. These will be helped/ensured by device-agent.
//
// 1. A information exchange between end-user and NAIS device administrator/slackbot:
// If BootstrapTokenPath is not present, user will be prompted to enroll using a generated token, and the agent will exit.
// When device-agent detects `BootstrapTokenPath` is present,
// it will generate a WireGuard config file called wg0.conf placed in `cfg.ConfigDir`
//
// 2. (When) A valid WireGuard config is present, ensure tunnel is configured and connected:
// - launch wireguard-go with the provided `cfg.Interface`, and run the following commands as root:
// - wg setconf `cfg.Interface` /etc/wireguard/wg0.conf
// - ifconfig `cfg.Interface` inet "`BootstrapConfig.TunnelIP`/21" "`BootstrapConfig.TunnelIP`" add
// - ifconfig `cfg.Interface` mtu 1380
// - ifconfig `cfg.Interface` up
// - route -q -n add -inet "`BootstrapConfig.TunnelIP`/21" -interface `cfg.Interface`
//
// 3. When connection is established:
// loop:
// Fetch device config from APIServer and configure gateways/generate WireGuard config
// loop:
// Monitor all connections, if one starts failing, re-fetch config and reset timer
func main() {
	log.Infof("starting device-agent with config:\n%+v", cfg)

	if err := filesExist(cfg.WireGuardPath, cfg.WireGuardGoBinary); err != nil {
		log.Fatalf("verifying if file exists: %v", err)
	}

	if err := ensureDirectories(cfg.ConfigDir, cfg.BinaryDir); err != nil {
		log.Fatalf("ensuring directory exists: %v", err)
	}

	if err := ensureKey(cfg.PrivateKeyPath, cfg.WireGuardPath); err != nil {
		log.Fatalf("ensuring private key exists: %v", err)
	}

	serial, err := getDeviceSerial()
	if err != nil {
		log.Fatalf("getting device serial: %v", err)
	}

	if err := filesExist(cfg.BootstrapTokenPath); err != nil {
		pubkey, err := generatePublicKey(cfg.PrivateKeyPath, cfg.WireGuardPath)
		if err != nil {
			log.Fatalf("generate public key during bootstrap: %v", err)
		}

		enrollmentToken, err := generateEnrollmentToken(serial, pubkey)
		if err != nil {
			log.Fatalf("generating enrollment token: %v", err)
		}

		fmt.Printf("no bootstrap token present. Send 'Nais Device' this message on slack: 'enroll %v'", enrollmentToken)
		os.Exit(0)
	}

	bootstrapToken, err := ioutil.ReadFile(cfg.BootstrapTokenPath)
	if err != nil {
		log.Fatalf("reading bootstrap token: %v", err)
	}

	cfg.BootstrapConfig, err = ParseBootstrapToken(string(bootstrapToken))
	if err != nil {
		log.Fatalf("parsing bootstrap config: %v", err)
	}

	if err := setupInterface(cfg); err != nil {
		log.Fatalf("setting up interface: %v", err)
	}

	privateKey, err := ioutil.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		log.Fatalf("Reading private key: %v", err)
	}

	baseConfig := GenerateBaseConfig(cfg.BootstrapConfig, privateKey)
	if err := actuateWireGuardConfig(baseConfig, cfg); err != nil {
		log.Fatalf("Actuating WireGuard config: %v", err)
	}

	for range time.NewTicker(10 * time.Second).C {
		gateways, err := getGatewayConfig(cfg.APIServer, serial)
		if err != nil {
			log.Errorf("Unable to get gateway config: %v", err)
		}

		wireGuardConfig := fmt.Sprintf("%s%s", baseConfig, gateways)

		if err := actuateWireGuardConfig(wireGuardConfig, cfg); err != nil {
			log.Errorf("Actuating WireGuard config: %v", err)
		}
	}
}

func getGatewayConfig(apiServerURL, serial string) (string, error) {
	deviceConfigAPI := fmt.Sprintf("%s/devices/config/%s", apiServerURL, serial)
	r, err := http.Get(deviceConfigAPI)
	if err != nil {
		return "", fmt.Errorf("getting device config: %w", err)
	}
	defer r.Body.Close()

	var gateways []Gateway
	if err := json.NewDecoder(r.Body).Decode(&gateways); err != nil {
		return "", fmt.Errorf("unmarshalling response body into gateways: %w", err)
	}

	return GenerateWireGuardPeers(gateways), nil
}

func GenerateWireGuardPeers(gateways []Gateway) string {
	peerTemplate := `
[Peer]
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

// actuateWireGuardConfig runs syncconfig with the provided WireGuard config
func actuateWireGuardConfig(wireGuardConfig string, config Config) error {
	if err := ioutil.WriteFile(cfg.WireGuardConfigPath, []byte(wireGuardConfig), 0600); err != nil {
		return fmt.Errorf("writing WireGuard config to disk: %w", err)
	}

	cmd := exec.Command(cfg.WireGuardPath, "syncconf", cfg.Interface, cfg.WireGuardConfigPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running syncconf: %w", err)
	}

	log.Debugf("Actuated WireGuard config: %v", cfg.WireGuardConfigPath)

	return nil
}

func generateEnrollmentToken(serial, publicKey string) (string, error) {
	type enrollmentConfig struct {
		Serial    string `json:"serial"`
		PublicKey string `json:"publicKey"`
	}

	ec := enrollmentConfig{
		Serial:    serial,
		PublicKey: publicKey,
	}

	if b, err := json.Marshal(ec); err != nil {
		return "", fmt.Errorf("marshalling enrollment config: %w", err)
	} else {
		return base64.StdEncoding.EncodeToString(b), nil
	}
}

// TODO(jhrv): extract this as a separate interface, with platform specific implmentations
func getDeviceSerial() (string, error) {
	cmd := exec.Command("/usr/sbin/ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	b, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting serial with ioreg: %w", err)
	}

	re := regexp.MustCompile("\"IOPlatformSerialNumber\" = \"([^\"]+)\"")
	matches := re.FindSubmatch(b)

	if len(matches) != 2 {
		return "", fmt.Errorf("unable to extract serial from output: %v", string(b))
	}

	return string(matches[1]), nil
}

func setupInterface(cfg Config) error {
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

	ip := cfg.BootstrapConfig.DeviceIP
	commands := [][]string{
		{cfg.WireGuardGoBinary, cfg.Interface},
		{"ifconfig", cfg.Interface, "inet", ip + "/21", ip, "add"},
		{"ifconfig", cfg.Interface, "mtu", "1380"},
		{"ifconfig", cfg.Interface, "up"},
		{"route", "-q", "-n", "add", "-inet", ip + "/21", "-interface", cfg.Interface},
	}

	return run(commands)
}

type BootstrapConfig struct {
	DeviceIP    string `json:"deviceIP"`
	PublicKey   string `json:"publicKey"`
	Endpoint    string `json:"tunnelEndpoint"`
	APIServerIP string `json:"apiServerIP"`
}

func ParseBootstrapToken(bootstrapToken string) (*BootstrapConfig, error) {
	b, err := base64.StdEncoding.DecodeString(bootstrapToken)
	if err != nil {
		return nil, fmt.Errorf("base64 decoding bootstrap token: %w", err)
	}

	var bootstrapConfig BootstrapConfig
	if err := json.Unmarshal(b, &bootstrapConfig); err != nil {
		return nil, fmt.Errorf("unmarshalling bootstrap token json: %w", err)
	}

	return &bootstrapConfig, nil
}

func GenerateBaseConfig(bootstrapConfig *BootstrapConfig, privateKey []byte) string {
	template := `
[Interface]
PrivateKey = %s

[Peer]
PublicKey = %s
AllowedIPs = %s/32
Endpoint = %s
`
	return fmt.Sprintf(template, privateKey, bootstrapConfig.PublicKey, bootstrapConfig.APIServerIP, bootstrapConfig.Endpoint)
}

func generatePublicKey(privateKeyPath string, wireguardPath string) (string, error) {
	cmd := exec.Command(wireguardPath, "pubkey")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("creating stdin pipe on 'wg pubkey': %w", err)
	}

	b, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("reading private key: %w", err)
	}

	if _, err := stdin.Write(b); err != nil {
		return "", fmt.Errorf("piping private key to 'wg genkey': %w", err)
	}

	if err := stdin.Close(); err != nil {
		return "", fmt.Errorf("closing stdin: %w", err)
	}

	b, err = cmd.Output()
	pubkey := strings.TrimSuffix(string(b), "\n")
	return pubkey, err
}

func filesExist(files ...string) error {
	for _, file := range files {
		if err := FileMustExist(file); err != nil {
			return err
		}
	}

	return nil
}

func ensureDirectories(dirs ...string) error {
	for _, dir := range dirs {
		if err := ensureDirectory(dir); err != nil {
			return err
		}
	}

	return nil
}

func ensureDirectory(dir string) error {
	info, err := os.Stat(dir)

	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0700)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%v is a file", dir)
	}

	return nil
}

func ensureKey(keyPath string, wireGuardPath string) error {
	if err := FileMustExist(keyPath); os.IsNotExist(err) {
		cmd := exec.Command(wireGuardPath, "genkey")
		stdout, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("executing %w: %v", err, string(stdout))
		}

		return ioutil.WriteFile(keyPath, stdout, 0600)
	} else if err != nil {
		return err
	}

	return nil
}

func DefaultConfig() Config {
	return Config{
		APIServer: "http://apiserver.device.nais.io",
		Interface: "utun69",
		ConfigDir: "/usr/local/etc/nais-device",
		BinaryDir: "/usr/local/bin",
		LogLevel:  "info",
	}
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
