package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
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
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
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
	gatewayChan := make(chan map[string]*apiserver.Gateway)

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
func connect(ctx context.Context, disconnectChan chan bool, gatewayChan chan map[string]*apiserver.Gateway) {
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

	for _, gateway := range rc.Gateways {
		err := ping(gateway.IP)
		if err != nil {
			gateway.SetHealthy(false)
		} else {
			gateway.SetHealthy(true)
		}
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

func ping(addr string) error {
	const (
		ProtocolICMP = 1
	)
	var ListenAddr = "0.0.0.0"
	// Start listening for icmp replies
	c, err := icmp.ListenPacket("ip4:icmp", ListenAddr)
	if err != nil {
		return err
	}
	defer c.Close()
	m := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1, //<< uint(seq), // TODO
			Data: []byte(""),
		},
	}
	b, err := m.Marshal(nil)
	if err != nil {
		return err
	}

	// Send it
	dst, err := net.ResolveIPAddr("ip4", addr)
	n, err := c.WriteTo(b, dst)
	if err != nil {
		return err
	} else if n != len(b) {
		return fmt.Errorf("got %v; want %v", n, len(b))
	}
	// Wait for a reply
	reply := make([]byte, 200)
	err = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return err
	}
	n, peer, err := c.ReadFrom(reply)
	if err != nil {
		return err
	}

	// Pack it up boys, we're done here
	rm, err := icmp.ParseMessage(ProtocolICMP, reply[:n])
	if err != nil {
		return err
	}
	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		return nil
	default:
		return fmt.Errorf("got %+v from %v; want echo reply", rm, peer)
	}
}
