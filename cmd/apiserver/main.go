package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nais/device/apiserver/api"
	"github.com/nais/device/apiserver/config"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/apiserver/slack"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = config.DefaultConfig()
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	flag.StringVar(&cfg.DbConnURI, "db-connection-uri", os.Getenv("DB_CONNECTION_URI"), "database connection URI (DSN)")
	flag.StringVar(&cfg.SlackToken, "slack-token", os.Getenv("SLACK_TOKEN"), "Slack token")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "Path to configuration directory")
	flag.StringVar(&cfg.Endpoint, "endpoint", cfg.Endpoint, "public endpoint (ip:port)")
	flag.BoolVar(&cfg.SkipSetupInterface, "skip-setup-interface", cfg.SkipSetupInterface, "Skip setting up WireGuard interface")

	flag.Parse()

	cfg.PrivateKeyPath = filepath.Join(cfg.ConfigDir, "private.key")
	cfg.WireGuardConfigPath = filepath.Join(cfg.ConfigDir, "wg0.conf")
}

func main() {
	if !cfg.SkipSetupInterface {
		if err := setupInterface(); err != nil {
			log.Fatalf("setting up WireGuard interface: %v", err)
		}
	}

	db, err := database.New(cfg.DbConnURI)
	if err != nil {
		log.Fatalf("instantiating database: %s", err)
	}

	if len(cfg.SlackToken) > 0 {
		slackBot := slack.New(cfg.SlackToken, cfg.Endpoint, db)
		go slackBot.Handler()
	}

	if !cfg.SkipSetupInterface {
		go syncWireguardConfig()
	}

	router := api.New(api.Config{DB: db})

	fmt.Println("running @", cfg.BindAddress)
	fmt.Println(http.ListenAndServe(cfg.BindAddress, router))
}

func setupInterface() error {
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
		{"ip", "address", "add", "dev", "wg0", "10.255.240.1/21"},
		{"ip", "link", "set", "wg0", "up"},
	}

	return run(commands)
}

func syncWireguardConfig() {
	db, err := database.New(cfg.DbConnURI)
	if err != nil {
		log.Fatalf("instantiating database: %v", err)
	}
	privateKey, err := ioutil.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		log.Fatalf("reading private key: %w", err)
	}

	for c := time.Tick(10 * time.Second); ; <-c {
		log.Info("syncing config")
		devices, err := db.ReadDevices()
		if err != nil {
			log.Errorf("reading devices from database: %v", err)
		}

		gateways, err := db.ReadGateways()
		if err != nil {
			log.Errorf("reading gateways from database: %v", err)
		}

		wgConfigContent := GenerateWGConfig(devices, gateways, privateKey)

		if err := ioutil.WriteFile(cfg.WireGuardConfigPath, wgConfigContent, 0600); err != nil {
			log.Errorf("writing WireGuard config to disk: %v", err)
		}

		if b, err := exec.Command("wg", "syncconf", "wg0", cfg.WireGuardConfigPath).Output(); err != nil {
			log.Errorf("synchronizing WireGuard config: %v: %v", err, string(b))
		}
	}
}

func GenerateWGConfig(devices []database.Device, gateways []database.Gateway, privateKey []byte) []byte {
	interfaceTemplate := `[Interface]
PrivateKeyPath = %s
ListenPort = 51820`

	wgConfig := fmt.Sprintf(interfaceTemplate, strings.TrimSuffix(string(privateKey), "\n"))

	peerTemplate := `
[Peer]
AllowedIPs = %s/32
PublicKey = %s`

	for _, device := range devices {
		wgConfig += fmt.Sprintf(peerTemplate, device.IP, device.PublicKey)
	}

	for _, gateway := range gateways {
		wgConfig += fmt.Sprintf(peerTemplate, gateway.IP, gateway.PublicKey)
	}

	return []byte(wgConfig)
}
