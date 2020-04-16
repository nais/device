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
	flag.StringVar(&cfg.ControlPlaneEndpoint, "control-plane-endpoint", cfg.ControlPlaneEndpoint, "Control Plane public endpoint (ip:port)")
	flag.BoolVar(&cfg.SkipSetupInterface, "skip-setup-interface", cfg.SkipSetupInterface, "Skip setting up WireGuard control plane interface")

	flag.Parse()

	cfg.PrivateKeyPath = filepath.Join(cfg.ConfigDir, "wgctrl-private.key")
	cfg.ControlPlaneWGConfigPath = filepath.Join(cfg.ConfigDir, "wgctrl.conf")
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
		slack := slack.New(cfg.SlackToken, cfg.ControlPlaneEndpoint, db)
		go slack.Handler()
	}

	if !cfg.SkipSetupInterface {
		go syncWireguardConfig()
	}

	router := api.New(api.Config{DB: db})

	fmt.Println("running @", cfg.BindAddress)
	fmt.Println(http.ListenAndServe(cfg.BindAddress, router))
}

func setupInterface() error {
	if err := exec.Command("ip", "link", "del", "wgctrl").Run(); err != nil {
		log.Infof("pre-deleting control plane interface (ok if this fails): %v", err)
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
		{"ip", "link", "add", "dev", "wgctrl", "type", "wireguard"},
		{"ip", "link", "set", "wgctrl", "mtu", "1380"},
		{"ip", "address", "add", "dev", "wgctrl", "10.255.240.1/21"},
		{"ip", "link", "set", "wgctrl", "up"},
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
		peers, err := db.ReadPeers("control")
		if err != nil {
			log.Errorf("reading peers from database: %v", err)
		}

		wgConfigContent := GenerateWGConfig(peers, privateKey)

		if err := ioutil.WriteFile(cfg.ControlPlaneWGConfigPath, wgConfigContent, 0600); err != nil {
			log.Errorf("writing control plane wireguard config to disk: %v", err)
		}

		if b, err := exec.Command("wg", "syncconf", "wgctrl", cfg.ControlPlaneWGConfigPath).Output(); err != nil {
			log.Errorf("synchronizing control plane WireGuard config: %v: %v", err, string(b))
		}
	}
}

func GenerateWGConfig(peers []database.Peer, privateKey []byte) []byte {
	interfaceTemplate := `[Interface]
PrivateKey = %s
ListenPort = 51820`

	wgConfig := fmt.Sprintf(interfaceTemplate, strings.TrimSuffix(string(privateKey), "\n"))

	peerTemplate := `
[Peer]
AllowedIPs = %s/32
PublicKey = %s`

	for _, peer := range peers {
		wgConfig += fmt.Sprintf(peerTemplate, peer.IP, peer.PublicKey)
	}

	return []byte(wgConfig)
}
