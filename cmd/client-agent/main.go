package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg                        = DefaultConfig()
	WGBinary                   string
	WireguardGoBinary          string
	ControlPlanePrivateKeyPath string
	ControlPlaneWGConfigPath   string
	DataPlanePrivateKeyPath    string
	DataPlaneWGConfigPath      string
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
}

// client-agent is responsible for enabling the end-user to connect to it's permitted gateways.
// To be able to connect, a series of prerequisites must be in place. These will be helped/ensured by client-agent.
//
// 1. A information exchange between end-user and naisdevice administrator/slackbot:
// - The end-user will provide it's generated public key ($(wg pubkey < `ControlPlanePrivateKeyPath`))
// - The end-user will receive the control plane tunnel endpoint and public key.
// The received information will be persisted as `ControlPlaneInfoFile`.
// When client-agent detects `ControlPlaneInfoFile` is present,
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
// $$$$$$
func main() {
	log.Infof("starting client-agent with config:\n%+v", cfg)

	if err := filesExist(WGBinary, WireguardGoBinary); err != nil {
		log.Fatalf("verifying if file exists: %v", err)
	}

	if err := ensureDirectories(cfg.ConfigDir, cfg.BinaryDir); err != nil {
		log.Fatalf("ensuring directory exists: %w", err)
	}

	if err := filesExist(ControlPlaneWGConfigPath); err != nil {
		// no wgctrl.conf

		if err := enroll(); err != nil {
			log.Fatalf("enrolling device: %v", err)
		}

	}

	for range time.NewTicker(10 * time.Second).C {
		log.Info("titei")
	}
}

type EnrollmentConfig struct {
	ClientIP    string `json:"clientIP"`
	PublicKey   string `json:"publicKey"`
	Endpoint    string `json:"endpoint"`
	APIServerIP string `json:"apiServerIP"`
}

func enroll() error {
	if err := ensureKey(ControlPlanePrivateKeyPath); err != nil {
		return fmt.Errorf("ensuring private key for control plane exists: %v", err)
	}

	if len(cfg.EnrollmentToken) == 0 {
		pubkey, err := generatePublicKey(ControlPlanePrivateKeyPath)
		if err != nil {
			return fmt.Errorf("generate public key during enroll: %v", err)
		}

		return fmt.Errorf("no enrollment token present. Send 'Nais Device' this message on slack: 'enroll %v'", string(pubkey))
	}

	privateKey, err := ioutil.ReadFile(ControlPlanePrivateKeyPath)
	if err != nil {
		return fmt.Errorf("reading private key: %w", err)
	}

	enrollmentConfig, err := ParseEnrollmentToken(cfg.EnrollmentToken)
	if err != nil {
		return fmt.Errorf("parsing enrollment token: %w", err)
	}

	wgConfigContent := GenerateWGConfig(enrollmentConfig, privateKey)
	fmt.Println(wgConfigContent)

	if err := ioutil.WriteFile(ControlPlaneWGConfigPath, wgConfigContent, 0600); err != nil {
		return fmt.Errorf("writing control plane wireguard config to disk: %w", err)
	}

	/*
	   	cmd := exec.Command("/usr/bin/wg", "syncconf", "wgdata", "/etc/wireguard/wgdata.conf")
	   	if stdout, err := cmd.Output(); err != nil {
	   		return fmt.Errorf("executing %w: %v", err, string(stdout))
	   	}

	   	return nil
	   }
	*/

	// create config file

	return nil
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

func generatePublicKey(privateKeyPath string) ([]byte, error) {
	cmd := exec.Command(WGBinary, "pubkey")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe on 'wg pubkey': %w", err)
	}

	b, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key: %w", err)
	}

	if _, err := stdin.Write(b); err != nil {
		return nil, fmt.Errorf("piping private key to 'wg genkey': %w", err)
	}

	return cmd.Output()
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
