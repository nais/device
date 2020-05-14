package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/nais/device/apiserver/kekw"
	"github.com/nais/device/device-agent/serial"
	"golang.org/x/crypto/curve25519"

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
	oauth2Config        oauth2.Config
	Platform            string
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

	setPlatform(&cfg)
	setPlatformDefaults(&cfg)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := platformPrerequisites(cfg)
	if err != nil {
		log.Errorf("Verifying platform prerequisites: %v", err)
		return
	}

	if err := filesExist(cfg.WireGuardPath); err != nil {
		log.Errorf("Verifying if file exists: %v", err)
		return
	}

	if err := ensureDirectories(cfg.ConfigDir); err != nil {
		log.Errorf("Ensuring directory exists: %v", err)
		return
	}

	if err := ensureKey(cfg.PrivateKeyPath); err != nil {
		log.Errorf("Ensuring private key exists: %v", err)
		return
	}

	deviceSerial, err := serial.GetDeviceSerial()
	if err != nil {
		log.Errorf("Getting device serial: %v", err)
		return
	}

	token, err := runAuthFlow(ctx, cfg.oauth2Config)
	if err != nil {
		log.Errorf("Unable to get token for user: %v", err)
		return
	}

	privateKey, err := ioutil.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		log.Errorf("Reading private key: %v", err)
		return
	}

	if err := filesExist(cfg.BootstrapTokenPath); err != nil {
		enrollmentToken, err := GenerateEnrollmentToken(deviceSerial, cfg.Platform, wgPubKey(privateKey))
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

	baseConfig := GenerateBaseConfig(cfg.BootstrapConfig, privateKey)

	if err := ioutil.WriteFile(cfg.WireGuardConfigPath, []byte(baseConfig), 0600); err != nil {
		log.Errorf("Writing WireGuard config to disk: %v", err)
		return
	}

	log.Debugf("Wrote base WireGuard config to disk")

	fmt.Println("Starting device-agent-helper, you might be prompted for password")

	if err = runHelper(ctx, cfg); err != nil {
		log.Errorf("Running helper: %v", err)
		return
	}

	client := cfg.oauth2Config.Client(ctx, token)

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-interrupt:
			log.Info("Received interrupt, shutting down gracefully.")
			return

		case <-time.After(15 * time.Second):
			gateways, err := getGateways(client, cfg.APIServer, deviceSerial)
			if err != nil {
				log.Errorf("Unable to get gateway config: %v", err)
			}

			wireGuardPeers := GenerateWireGuardPeers(gateways)

			if err := ioutil.WriteFile(cfg.WireGuardConfigPath, []byte(baseConfig+wireGuardPeers), 0600); err != nil {
				log.Errorf("Writing WireGuard config to disk: %v", err)
				return
			}

			log.Debugf("Wrote WireGuard config to disk")
		}
	}
}

func runAuthFlow(ctx context.Context, conf oauth2.Config) (*oauth2.Token, error) {
	server := &http.Server{Addr: "127.0.0.1:51800"}

	// Ignoring impossible error
	codeVerifier, _ := codeverifier.CreateCodeVerifier()

	method := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	challenge := oauth2.SetAuthURLParam("code_challenge", codeVerifier.CodeChallengeS256())

	//TODO check this in response from Azure
	randomString := random.RandomString(16, random.LettersAndNumbers)

	tokenChan := make(chan *oauth2.Token)
	// define a handler that will get the authorization code, call the token endpoint, and close the HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			log.Errorf("Error: could not find 'code' URL query parameter")
			failureResponse(w, "Error: could not find 'code' URL query parameter")
			tokenChan <- nil
			return
		}

		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(30*time.Second))
		defer cancel()

		codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", codeVerifier.String())
		t, err := conf.Exchange(ctx, code, codeVerifierParam)
		if err != nil {
			failureResponse(w, "Error: during code exchange")
			log.Errorf("exchanging code for tokens: %v", err)
			tokenChan <- nil
			return
		}

		successfulResponse(w, "Successfully authenticated 👌 Close me pls")
		tokenChan <- t
	})

	go func() {
		_ = server.ListenAndServe()
	}()

	url := conf.AuthCodeURL(randomString, oauth2.AccessTypeOffline, method, challenge)
	command := exec.Command("open", url)
	_ = command.Start()
	fmt.Printf("If the browser didn't open, visit this url to sign in: %v\n", url)

	token := <-tokenChan
	_ = server.Close()

	if token == nil {
		return nil, fmt.Errorf("no token received")
	}

	return token, nil
}

func getGateways(client *http.Client, apiServerURL, serial string) ([]Gateway, error) {
	deviceConfigAPI := fmt.Sprintf("%s/devices/%s/gateways", apiServerURL, serial)
	r, err := client.Get(deviceConfigAPI)
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

func GenerateEnrollmentToken(serial, platform string, publicKey []byte) (string, error) {
	type enrollmentConfig struct {
		Serial    string `json:"serial"`
		PublicKey string `json:"publicKey"`
		Platform  string `json:"platform"`
	}

	ec := enrollmentConfig{
		Serial:    serial,
		PublicKey: string(KeyToBase64(publicKey)),
		Platform:  platform,
	}

	if b, err := json.Marshal(ec); err != nil {
		return "", fmt.Errorf("marshalling enrollment config: %w", err)
	} else {
		return base64.StdEncoding.EncodeToString(b), nil
	}
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

func ensureKey(keyPath string) error {
	if err := FileMustExist(keyPath); os.IsNotExist(err) {
		return ioutil.WriteFile(keyPath, KeyToBase64(WgGenKey()), 0600)
	} else if err != nil {
		return err
	}

	return nil
}

func DefaultConfig() Config {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Getting user conig dir: %w", err)
	}

	return Config{
		APIServer: "http://10.255.240.1",
		Interface: "utun69",
		ConfigDir: filepath.Join(userConfigDir, "nais-device"),
		BinaryDir: "/usr/local/bin",
		LogLevel:  "info",
		oauth2Config: oauth2.Config{
			ClientID:    "8086d321-c6d3-4398-87da-0d54e3d93967",
			Scopes:      []string{"openid", "6e45010d-2637-4a40-b91d-d4cbb451fb57/.default", "offline_access"},
			Endpoint:    endpoints.AzureAD("62366534-1ec3-4962-8869-9b5535279d0b"),
			RedirectURL: "http://localhost:51800",
		},
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

func getHomeDirOrFail() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Getting home dir: %v", err)
	}

	return homeDir
}

func failureResponse(w http.ResponseWriter, msg string) {
	w.Header().Set("content-type", "text/html;charset=utf8")
	_, _ = fmt.Fprintf(w, `
<h2>
  %s
</h2>
<img width="100" src="data:image/jpeg;base64,%s"/>
`, msg, kekw.SadKekW)
}

func successfulResponse(w http.ResponseWriter, msg string) {
	w.Header().Set("content-type", "text/html;charset=utf8")
	_, _ = fmt.Fprintf(w, `
<h2>
  %s
</h2>
<img width="100" src="data:image/jpeg;base64,%s"/>
`, msg, kekw.HappyKekW)
}

func adminCommandContext(ctx context.Context, command string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "sudo", append([]string{command}, arg...)...)
}

func WireGuardGenerateKeyPair() (string, string) {
	var publicKeyArray [32]byte
	var privateKeyArray [32]byte

	n, err := rand.Read(privateKeyArray[:])

	if err != nil || n != len(privateKeyArray) {
		panic("Unable to generate random bytes")
	}

	privateKeyArray[0] &= 248
	privateKeyArray[31] = (privateKeyArray[31] & 127) | 64

	curve25519.ScalarBaseMult(&publicKeyArray, &privateKeyArray)

	publicKeyString := base64.StdEncoding.EncodeToString(publicKeyArray[:])
	privateKeyString := base64.StdEncoding.EncodeToString(privateKeyArray[:])

	return publicKeyString, privateKeyString
}

func KeyToBase64(key []byte) []byte {
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(key)))
	base64.StdEncoding.Encode(dst, key)
	return dst
}

func WgGenKey() []byte {
	var privateKey [32]byte

	n, err := rand.Read(privateKey[:])

	if err != nil || n != len(privateKey) {
		panic("Unable to generate random bytes")
	}

	privateKey[0] &= 248
	privateKey[31] = (privateKey[31] & 127) | 64
	return privateKey[:]
}

func wgPubKey(privateKeySlice []byte) []byte {
	var privateKey [32]byte
	var publicKey [32]byte
	copy(privateKeySlice[:], privateKey[:])

	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return publicKey[:]
}
