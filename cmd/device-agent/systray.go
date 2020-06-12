package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/config"
	"github.com/nais/device/device-agent/filesystem"
	"github.com/nais/device/device-agent/runtimeconfig"
	"github.com/nais/device/device-agent/wireguard"
	log "github.com/sirupsen/logrus"
)

var (
	cfg = config.DefaultConfig()
)

func onReady() {
	currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	iconPath := filepath.Join(currentDir, "..", "..", "assets", "naislogo.png")
	icon, err := ioutil.ReadFile(iconPath)

	if err != nil {
		fmt.Errorf("unable to find the icon")
	}
	systray.SetIcon(icon)

	cfg.SetDefaults()
	if err = filesystem.EnsurePrerequisites(&cfg); err != nil {
		beeep.Alert("Warning", fmt.Sprintf("Missing prerequisites: %s", err), iconPath)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	disconnectChan := make(chan bool, 1)
	gatewayChan := make(chan []apiserver.Gateway)

	mConnect := systray.AddMenuItem("Connect", "Bootstrap the nais device")
	mQuit := systray.AddMenuItem("Quit", "exit the application")
	systray.AddSeparator()
	mCurrentGateways := make(map[string]*systray.MenuItem)
	go func() {
		connected := false
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for {
			select {
			case <-mConnect.ClickedCh:
				if connected {
					mConnect.SetTitle("Connect")
					connected = false
					disconnectChan <- true
				} else {
					mConnect.SetTitle("Disconnect")
					connected = true
					go connect(ctx, disconnectChan, gatewayChan)
				}
			case <-mQuit.ClickedCh:
				log.Info("Exiting")
				systray.Quit()
			case <-interrupt:
				log.Info("Received interrupt, shutting down gracefully.")
				disconnectChan <- true
				time.Sleep(time.Second * 2)
				systray.Quit()
			case gateways := <-gatewayChan:
				for _, gateway := range gateways {
					if _, ok := mCurrentGateways[gateway.Endpoint]; !ok {
						mCurrentGateways[gateway.Endpoint] = systray.AddMenuItem(gateway.Name, gateway.Endpoint)
						mCurrentGateways[gateway.Endpoint].Disable()
					}
				}

			}
		}
	}()

}
func onExit() {
	// This is where we clean up
}
func connect(ctx context.Context, disconnectChan chan bool, gatewayChan chan []apiserver.Gateway) {
	rc, err := runtimeconfig.New(cfg, ctx)
	if err != nil {
		log.Fatalf("Initializing runtime config: %v", err)
	}

	baseConfig := wireguard.GenerateBaseConfig(rc.BootstrapConfig, rc.PrivateKey)

	if err := ioutil.WriteFile(cfg.WireGuardConfigPath, []byte(baseConfig), 0600); err != nil {
		log.Fatalf("Writing base WireGuard config to disk: %v", err)
	}

	// wait until helper has established tunnel to apiserver...
	if rc.SessionInfo, err = ensureValidSessionInfo(cfg.SessionInfoPath, cfg.APIServer, cfg.Platform, rc.Serial, ctx); err != nil {
		log.Errorf("Ensuring valid session key: %v", err)
		return
	}

	if err := writeToJSONFile(rc.SessionInfo, cfg.SessionInfoPath); err != nil {
		log.Errorf("Writing session info to disk: %v", err)
		return
	}

ConnectedLoop:
	for {
		select {
		case <-disconnectChan:
			log.Info("Disconnecting")
			break ConnectedLoop
		case <-time.After(5 * time.Second):
			if err := SyncConfig(baseConfig, rc, ctx); err != nil {
				log.Errorf("Unable to synchronize config with apiserver: %v", err)
			}
			gatewayChan <- rc.Gateways
		}
	}
}
