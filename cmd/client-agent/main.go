package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	cfg                        = DefaultConfig()
	WGBinary                   string
	WireguardGoBinary          string
	ControlPlanePrivateKeyPath string
	ControlPlaneWGConfigPath   string
	DataPlanePrivateKeyPath    string
	DataPlaneWGConfigPath      string
	EnrollmentTokenPath        string
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	flag.StringVar(&cfg.APIServer, "apiserver", cfg.APIServer, "hostname to apiserver")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "path to agent config directory")
	flag.StringVar(&cfg.BinaryDir, "binary-dir", cfg.BinaryDir, "path to binary directory")
	flag.StringVar(&cfg.ControlPlaneInterface, "control-plane-interface", cfg.ControlPlaneInterface, "name of control plane tunnel interface")
	flag.StringVar(&cfg.DataPlaneInterface, "data-plane-interface", cfg.DataPlaneInterface, "name of data plane tunnel interface")
	flag.StringVar(&cfg.EnrollmentToken, "enrollment-token", cfg.EnrollmentToken, "enrollment token")

	flag.Parse()

	WGBinary = filepath.Join(cfg.BinaryDir, "naisdevice-wg")
	WireguardGoBinary = filepath.Join(cfg.BinaryDir, "naisdevice-wireguard-go")
	ControlPlanePrivateKeyPath = filepath.Join(cfg.ConfigDir, "wgctrl-private.key")
	ControlPlaneWGConfigPath = filepath.Join(cfg.ConfigDir, "wgctrl.conf")
	DataPlanePrivateKeyPath = filepath.Join(cfg.ConfigDir, "wgdata-private.key")
	DataPlaneWGConfigPath = filepath.Join(cfg.ConfigDir, "wgdata.conf")
	EnrollmentTokenPath = filepath.Join(cfg.ConfigDir, "enrollment.token")
}

// client-agent is responsible for enabling the end-user to connect to it's permitted gateways.
// To be able to connect, a series of prerequisites must be in place. These will be helped/ensured by client-agent.
//

// 1. A information exchange between end-user and naisdevice administrator/slackbot:
// If neither EnrollmentTokenPath nor EnrollmentToken is present, user will be prompted to enroll using public key, and the agent will exit.
// If EnrollmentToken command line option is provided, the agent will store it as EnrollmentTokenPath.
// If EnrollmentTokenPath is present the agent will generate `wgctrl.conf` and continue.

// - The end-user will provide it's generated public key ($(wg pubkey < `ControlPlanePrivateKeyPath`))
// - The end-user will receive the control plane tunnel endpoint, public key, apiserver tunnel ip, and it's own tunnel
//   ip encoded as a base64 string.
// The received information will be persisted as `EnrollmentTokenPath`.
// When client-agent detects `EnrollmentTokenPath` is present,
// it will generate a WireGuard config file called wgctrl.conf placed in `cfg.ConfigDir`
//
// 2. (When) A valid control plane WireGuard config is present, ensure control plane tunnel is configured and connected:
// - launch wireguard-go with the provided `cfg.ControlPlaneInterface`, and run the following commands:
// - sudo wg setconf "$wgctrl_device" /etc/wireguard/wgctrl.conf
// - sudo ifconfig `cfg.ControlPlaneInterface` inet "`ControlPlaneInfoFile.TunnelIP`/21" "`ControlPlaneInfoFile.TunnelIP`" add
// - sudo ifconfig `cfg.ControlPlaneInterface` mtu 1380
// - sudo ifconfig `cfg.ControlPlaneInterface` up
// - sudo route -q -n add -inet "`ControlPlaneInfoFile.TunnelIP`/21" -interface "$wgctrl_device"
//
// 3.
//
// 'client-agent' binary is packaged as 'naisdevice-agent'
// alongside naisdevice-wg and naisdevice-wireguard-go (for MacOS and Windows)
// Binaries will be reside in /usr/local/bin
// runs as root
//TODO: detect cfg.ConfigDir/wgctrl.conf
//TODO: if missing, notify user ^
//TODO: establish ctrl plane
//TODO: (authenticate user, not part of MVP)
//TODO: get config from apiserver
//TODO: establish data plane (continously monitored, will trigger ^ if it goes down and user wants to connect)
// $$$$$$$$$
func main() {
	log.Infof("starting client-agent with config:\n%+v", cfg)

	if err := filesExist(WGBinary, WireguardGoBinary); err != nil {
		log.Fatalf("verifying if file exists: %v", err)
	}

	if err := ensureDirectories(cfg.ConfigDir, cfg.BinaryDir); err != nil {
		log.Fatalf("ensuring directory exists: %w", err)
	}

	if err := ensureKey(ControlPlanePrivateKeyPath); err != nil {
		log.Fatalf("ensuring private key for control plane exists: %v", err)
	}

	if len(cfg.EnrollmentToken) == 0 {
		if err := filesExist(EnrollmentTokenPath); err != nil {
			pubkey, err := generatePublicKey(ControlPlanePrivateKeyPath)
			if err != nil {
				log.Fatalf("generate public key during enroll: %v", err)
			}

			serial, err := getDeviceSerial()
			if err != nil {
				log.Fatalf("getting device serial: %v", err)
			}

			fmt.Printf("no enrollment token present. Send 'Nais Device' this message on slack: 'enroll %v %v'", serial, pubkey)
			os.Exit(0)
		}

		enrollmentToken, err := ioutil.ReadFile(EnrollmentTokenPath)
		if err != nil {
			log.Fatalf("reading enrollment token: %v", err)
		}

		cfg.EnrollmentToken = string(enrollmentToken)
	} else {
		if err := ioutil.WriteFile(EnrollmentTokenPath, []byte(cfg.EnrollmentToken), 0600); err != nil {
			log.Fatalf("writing enrollment token to disk: %v", err)
		}
	}

	if err := setupControlPlane(cfg.EnrollmentToken); err != nil {
		log.Fatalf("setting up control plane: %v", err)
	}

	for range time.NewTicker(10 * time.Second).C {
		log.Info("titei")
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

func setupControlPlane(enrollmentToken string) error {
	enrollmentConfig, err := ParseEnrollmentToken(enrollmentToken)
	if err != nil {
		return fmt.Errorf("parsing enrollment config key: %w", err)
	}

	privateKey, err := ioutil.ReadFile(ControlPlanePrivateKeyPath)
	if err != nil {
		return fmt.Errorf("reading private key: %w", err)
	}

	wgConfigContent := GenerateWGConfig(enrollmentConfig, privateKey)
	fmt.Println(string(wgConfigContent))

	if err := ioutil.WriteFile(ControlPlaneWGConfigPath, wgConfigContent, 0600); err != nil {
		return fmt.Errorf("writing control plane wireguard config to disk: %w", err)
	}

	if err := setupInterface(cfg.ControlPlaneInterface, enrollmentConfig.ClientIP, ControlPlaneWGConfigPath); err != nil {
		return fmt.Errorf("setting up control plane interface: %w", err)
	}

	return nil
}

func setupInterface(interfaceName string, ip string, configPath string) error {
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
		{WireguardGoBinary, interfaceName},
		{"ifconfig", interfaceName, "inet", ip + "/21", ip, "add"},
		{"ifconfig", interfaceName, "mtu", "1380"},
		{"ifconfig", interfaceName, "up"},
		{"route", "-q", "-n", "add", "-inet", ip + "/21", "-interface", interfaceName},
		{WGBinary, "syncconf", interfaceName, configPath},
	}

	return run(commands)
}

type EnrollmentConfig struct {
	ClientIP    string `json:"clientIP"`
	PublicKey   string `json:"publicKey"`
	Endpoint    string `json:"endpoint"`
	APIServerIP string `json:"apiServerIP"`
}

func ParseEnrollmentToken(enrollmentToken string) (enrollmentConfig *EnrollmentConfig, err error) {
	b, err := base64.StdEncoding.DecodeString(enrollmentToken)
	if err != nil {
		return nil, fmt.Errorf("base64 decoding enrollment token: %w", err)
	}

	if err := json.Unmarshal(b, &enrollmentConfig); err != nil {
		return nil, fmt.Errorf("unmarshalling enrollment token json: %w", err)
	}

	return
}

func GenerateWGConfig(enrollmentConfig *EnrollmentConfig, privateKey []byte) []byte {
	template := `
[Interface]
PrivateKey = %s

[Peer]
PublicKey = %s
AllowedIPs = %s
Endpoint = %s
`
	return []byte(fmt.Sprintf(template, privateKey, enrollmentConfig.PublicKey, enrollmentConfig.APIServerIP, enrollmentConfig.Endpoint))
}

func generatePublicKey(privateKeyPath string) (string, error) {
	cmd := exec.Command(WGBinary, "pubkey")

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

func ensureKey(keypath string) error {
	if err := FileMustExist(keypath); os.IsNotExist(err) {
		cmd := exec.Command(WGBinary, "genkey")
		stdout, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("executing %w: %v", err, string(stdout))
		}

		return ioutil.WriteFile(keypath, stdout, 0600)
	} else if err != nil {
		return err
	}

	return nil
}

type Config struct {
	APIServer             string
	DataPlaneInterface    string
	ControlPlaneInterface string
	ConfigDir             string
	BinaryDir             string
	EnrollmentToken       string
}

func DefaultConfig() Config {
	return Config{
		APIServer:             "http://apiserver.device.nais.io",
		DataPlaneInterface:    "utun34",
		ControlPlaneInterface: "utun35",
		ConfigDir:             "/usr/local/etc/nais-device",
		BinaryDir:             "/usr/local/bin",
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
