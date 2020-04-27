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

	"github.com/nais/device/pkg/random"
	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
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
	PublicKey string   `json:"publicKey"`
	Endpoint  string   `json:"endpoint"`
	IP        string   `json:"ip"`
	Routes    []string `json:"routes"`
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
// A information exchange between end-user and NAIS device administrator/slackbot:
// If BootstrapTokenPath is not present, user will be prompted to enroll using a generated token, and the agent will exit.
// When device-agent detects a valid bootstrap token, it will generate a WireGuard config file called wg0.conf placed in `cfg.ConfigDir`
// This file will initially only contain the Interface definition and the APIServer peer.
//
// It will run the device-agent-helper with params....
//
// loop:
// Fetch device config from APIServer and configure generate and write WireGuard config to disk
// loop:
// Monitor all connections, if one starts failing, re-fetch config and reset timer
func main() {
	log.Infof("Starting device-agent with config:\n%+v", cfg)

	if err := filesExist(cfg.WireGuardPath, cfg.WireGuardGoBinary); err != nil {
		log.Errorf("Verifying if file exists: %v", err)
		return
	}

	if err := ensureDirectories(cfg.ConfigDir, cfg.BinaryDir); err != nil {
		log.Errorf("Ensuring directory exists: %v", err)
		return
	}

	if err := ensureKey(cfg.PrivateKeyPath, cfg.WireGuardPath); err != nil {
		log.Errorf("Ensuring private key exists: %v", err)
		return
	}

	serial, err := getDeviceSerial()
	if err != nil {
		log.Errorf("Getting device serial: %v", err)
		return
	}

	//ensureAADToken()

	if err := filesExist(cfg.BootstrapTokenPath); err != nil {
		pubkey, err := generatePublicKey(cfg.PrivateKeyPath, cfg.WireGuardPath)
		if err != nil {
			log.Errorf("Generate public key during bootstrap: %v", err)
			return
		}

		enrollmentToken, err := generateEnrollmentToken(serial, pubkey)
		if err != nil {
			log.Errorf("Generating enrollment token: %v", err)
			return
		}

		fmt.Printf("\n---\nno bootstrap token present. Send 'naisdevice' your enrollment token on slack by copying and pasting this:\n/msg @naisdevice enroll %v\n\n", enrollmentToken)
		return
	}

	bootstrapToken, err := ioutil.ReadFile(cfg.BootstrapTokenPath)
	if err != nil {
		log.Errorf("Reading bootstrap token: %v", err)
		return
	}

	cfg.BootstrapConfig, err = ParseBootstrapToken(string(bootstrapToken))
	if err != nil {
		log.Errorf("Parsing bootstrap config: %v", err)
		return
	}

	privateKey, err := ioutil.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		log.Errorf("Reading private key: %v", err)
		return
	}

	baseConfig := GenerateBaseConfig(cfg.BootstrapConfig, privateKey)

	if err := ioutil.WriteFile(cfg.WireGuardConfigPath, []byte(baseConfig), 0600); err != nil {
		log.Errorf("Writing WireGuard config to disk: %w", err)
		return
	}

	log.Debugf("Wrote base WireGuard config to disk")

	fmt.Println("Starting device-agent-helper, you might be prompted for password")
	// start device-agent-helper
	cmd := exec.Command("sudo", "./bin/device-agent-helper",
		"--interface", cfg.Interface,
		"--tunnel-ip", cfg.BootstrapConfig.TunnelIP,
		"--wireguard-binary", cfg.WireGuardPath,
		"--wireguard-go-binary", cfg.WireGuardGoBinary,
		"--wireguard-config-path", cfg.WireGuardConfigPath)

	if err := cmd.Start(); err != nil {
		log.Errorf("Starting device-agent-helper: %v", err)
		return
	}

	//TODO(VegarM): defer context abort

	for range time.NewTicker(10 * time.Second).C {
		gateways, err := getGateways(cfg.APIServer, serial)
		if err != nil {
			log.Errorf("Unable to get gateway config: %v", err)
		}

		wireGuardPeers := GenerateWireGuardPeers(gateways)

		if err := ioutil.WriteFile(cfg.WireGuardConfigPath, []byte(baseConfig+wireGuardPeers), 0600); err != nil {
			log.Errorf("Writing WireGuard config to disk: %w", err)
			return
		}

		log.Debugf("Wrote WireGuard config to disk")
	}
}

func ensureAADToken() {
	redirectURL := "http://localhost:51800"

	//ctx := context.Background()

	conf := &oauth2.Config{
		ClientID:    "5d69cfe1-b300-4a1a-95ec-4752d07ccab1",
		Scopes:      []string{"openid", "5d69cfe1-b300-4a1a-95ec-4752d07ccab1/.default", "offline_access"},
		Endpoint:    endpoints.AzureAD("62366534-1ec3-4962-8869-9b5535279d0b"),
		RedirectURL: redirectURL,
	}

	//server := &http.Server{Addr: ":1337"}

	// Redirect user to consent page to ask for permission
	// for the scopes specified above.

	// Ignoring impossible error
	codeVerifier, _ := codeverifier.CreateCodeVerifier()

	method := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	challenge := oauth2.SetAuthURLParam("code_challenge", codeVerifier.CodeChallengeS256())

	url := conf.AuthCodeURL(random.RandomString(16, random.LettersAndNumbers), oauth2.AccessTypeOffline, method, challenge)

	// macos: $ open $url

	command := exec.Command("open", url)
	command.Start()
	fmt.Printf("If the browser didn't open, visit this url to sign in: %v\n", url)

	//// Use the authorization code that is pushed to the redirect
	//// URL. Exchange will do the handshake to retrieve the
	//// initial access token. The HTTP Client returned by
	//// conf.Client will refresh the token as necessary.
	//
	//// define a handler that will get the authorization code, call the token endpoint, and close the HTTP server
	//http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	//	// get the authorization code
	//	code := r.URL.Query().Get("code")
	//	if code == "" {
	//		fmt.Println("snap: Url Param 'code' is missing")
	//		io.WriteString(w, "Error: could not find 'code' URL parameter\n")
	//
	//		return
	//	}
	//
	//	codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", codeVerifier.String())
	//	tok, err := conf.Exchange(ctx, code, codeVerifierParam)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//
	//	io.WriteString(w, "hei\n")
	//	fmt.Println("Successfully logged in.")
	//
	//	client := conf.Client(ctx, tok)
	//	client.Get("...")
	//})
	//
	//server.ListenAndServe()
	//defer server.Close()
}

func getGateways(apiServerURL, serial string) ([]Gateway, error) {
	deviceConfigAPI := fmt.Sprintf("%s/devices/config/%s", apiServerURL, serial)
	r, err := http.Get(deviceConfigAPI)
	if err != nil {
		return nil, fmt.Errorf("getting device config: %w", err)
	}
	defer r.Body.Close()

	var gateways []Gateway
	if err := json.NewDecoder(r.Body).Decode(&gateways); err != nil {
		return nil, fmt.Errorf("unmarshalling response body into gateways: %w", err)
	}

	return gateways, nil
}

func setupRoutes(gateways []Gateway, interfaceName string) error {
	for _, gateway := range gateways {
		for _, route := range gateway.Routes {
			cmd := exec.Command("route", "-q", "-n", "add", "-inet", route, "-interface", interfaceName)
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Errorf("%v: %v", cmd, string(output))
				return fmt.Errorf("executing %v: %w", cmd, err)
			}
			log.Debugf("%v: %v", cmd, string(output))
		}
	}
	return nil
}

func GenerateWireGuardPeers(gateways []Gateway) string {
	peerTemplate := `[Peer]
PublicKey = %s
AllowedIPs = %s
Endpoint = %s
`
	var peers string

	for _, gateway := range gateways {
		allowedIPs := strings.Join(append(gateway.Routes, gateway.IP+"/32"), ",")
		peers += fmt.Sprintf(peerTemplate, gateway.PublicKey, allowedIPs, gateway.Endpoint)
	}

	return peers
}

// writeWireGuardConfig runs syncconfig with the provided WireGuard config
func writeWireGuardConfig(wireGuardConfig string, config Config) error {
	if err := ioutil.WriteFile(cfg.WireGuardConfigPath, []byte(wireGuardConfig), 0600); err != nil {
		return fmt.Errorf("writing WireGuard config to disk: %w", err)
	}

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
	b, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("getting serial with ioreg: %w: %v", err, string(b))
	}

	re := regexp.MustCompile("\"IOPlatformSerialNumber\" = \"([^\"]+)\"")
	matches := re.FindSubmatch(b)

	if len(matches) != 2 {
		return "", fmt.Errorf("unable to extract serial from output: %v", string(b))
	}

	return string(matches[1]), nil
}

type BootstrapConfig struct {
	TunnelIP    string `json:"deviceIP"`
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
	template := `[Interface]
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
		stdout, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("executing command: %v, %w: %v", cmd, err, string(stdout))
		}

		return ioutil.WriteFile(keyPath, stdout, 0600)
	} else if err != nil {
		return err
	}

	return nil
}

func DefaultConfig() Config {
	return Config{
		APIServer: "http://10.255.240.1",
		Interface: "utun69",
		ConfigDir: "~/.config/nais-device",
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
