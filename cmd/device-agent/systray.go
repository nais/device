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
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type ProgramState int

const (
	StateDisconnected ProgramState = iota
	StateBootstrapping
	StateConnecting
	StateConnected
	StateDisconnecting
	StateQuitting
)

type GuiState struct {
	ProgramState ProgramState
	Gateways     apiserver.Gateways
}

func (g GuiState) String() string {
	switch g.ProgramState {
	case StateDisconnected:
		return "Disconnected"
	case StateBootstrapping:
		return "Bootstrapping..."
	case StateConnecting:
		return "Connecting..."
	case StateConnected:
		return "Connected"
	case StateDisconnecting:
		return "Disconnecting..."
	case StateQuitting:
		return "Quitting..."
	default:
		return "Unknown state!!!"
	}
}

var (
	cfg      = config.DefaultConfig()
	state    = StateDisconnected
	newstate = make(chan ProgramState, 1)
)

func notify(message string) {
	err := beeep.Notify("NAIS device", message, "")
	log.Infof("sending message to notification centre: %s", message)
	if err != nil {
		log.Errorf("failed sending message due to error: %s", err)
	}
}

// read in external gui events
func guiloop(mConnect, mQuit *systray.MenuItem, interrupt chan os.Signal) {
	for {
		select {
		case <-mConnect.ClickedCh:
			if state == StateDisconnected {
				newstate <- StateConnecting
			} else {
				newstate <- StateDisconnecting
			}
		case <-mQuit.ClickedCh:
			newstate <- StateQuitting
		case <-interrupt:
			log.Info("Received interrupt, shutting down gracefully.")
			newstate <- StateQuitting
		}
	}
}

func mainloop(updateGUI func(guiState GuiState)) {
	var rc *runtimeconfig.RuntimeConfig
	var err error

	stop := make(chan interface{}, 1)

	for st := range newstate {
		state = st

		//noinspection GoNilness
		updateGUI(GuiState{
			ProgramState: state,
			Gateways:     rc.GetGateways(),
		})

		switch state {
		case StateDisconnected:
		case StateBootstrapping:
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute*3)
			rc, err = runtimeconfig.New(cfg, ctx)
			cancel()
			if err != nil {
				notify(err.Error())
				newstate <- StateDisconnected
				continue
			}
			err = WriteConfigFile(rc.Config.WireGuardConfigPath, *rc)
			if err != nil {
				err = fmt.Errorf("unable to write WireGuard configuration file: %w", err)
				notify(err.Error())
				newstate <- StateDisconnected
				continue
			}
			newstate <- StateConnecting

		case StateConnecting:
			if rc == nil {
				newstate <- StateBootstrapping
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			err := connect(ctx, rc)
			cancel()

			if err == nil {
				newstate <- StateConnected
			}

		case StateConnected:
			go connectedLoop(stop, rc)

		case StateDisconnecting:
			stop <- new(interface{})
			newstate <- StateDisconnected

		case StateQuitting:
			stop <- new(interface{})
			systray.Quit()
		}
	}
}

func connectedLoop(stop chan interface{}, rc *runtimeconfig.RuntimeConfig) {
	timeout := 5 * time.Second
	ticker := time.NewTicker(timeout)

	for {
		select {
		case <-ticker.C:

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			gateways, err := apiserver.GetGateways(rc.SessionInfo.Key, rc.Config.APIServer, ctx)
			cancel()

			if err != nil {
				log.Errorf("unable to get gateway config: %v", err)
				continue
			}

			if ue, ok := err.(*apiserver.UnauthorizedError); ok {
				newstate <- StateDisconnecting
				log.Errorf("unauthorized access from apiserver: %v", ue)
				log.Errorf("assuming invalid session; disconnecting.")
				continue
			}

			for _, gw := range gateways {
				err := ping(gw.IP)
				if err == nil {
					gw.Healthy = true
				} else {
					gw.Healthy = false
					log.Errorf("unable to ping host %s: %v", gw.IP, err)
				}
			}

			rc.Gateways = gateways

			err = WriteConfigFile(rc.Config.WireGuardConfigPath, *rc)
			if err != nil {
				err = fmt.Errorf("unable to write WireGuard configuration file: %w", err)
				notify(err.Error())
			}

		case <-stop:
			return
		}
	}
}

func onReady() {
	currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	iconPath := filepath.Join(currentDir, "..", "..", "assets", "naislogo.png")
	icon, err := ioutil.ReadFile(iconPath)

	if err != nil {
		log.Errorf("unable to find the icon")
	} else {
		systray.SetIcon(icon)
	}

	cfg.SetDefaults()
	if err = filesystem.EnsurePrerequisites(&cfg); err != nil {
		notify(fmt.Sprintf("Missing prerequisites: %s", err))
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	mState := systray.AddMenuItem("", "State")
	mState.Disable()
	mConnect := systray.AddMenuItem("Connect", "Bootstrap the nais device")
	mQuit := systray.AddMenuItem("Quit", "exit the application")
	systray.AddSeparator()
	mCurrentGateways := make(map[string]*systray.MenuItem)

	updateGUI := func(st GuiState) {
		mState.SetTitle(st.String())
		switch st.ProgramState {
		case StateDisconnected:
			mConnect.SetTitle("Connect")
			mConnect.Enable()
		case StateConnected:
			mConnect.SetTitle("Disconnect")
			mConnect.Enable()
		default:
			mConnect.Disable()
		}
		for _, gateway := range st.Gateways {
			if _, ok := mCurrentGateways[gateway.Endpoint]; !ok {
				mCurrentGateways[gateway.Endpoint] = systray.AddMenuItem(gateway.Name, gateway.Endpoint)
				mCurrentGateways[gateway.Endpoint].Disable()
			}
		}
	}

	go guiloop(mConnect, mQuit, interrupt)
	newstate <- StateDisconnected
	mainloop(updateGUI)
}

func onExit() {
	// This is where we clean up
}

func connect(ctx context.Context, rc *runtimeconfig.RuntimeConfig) error {
	var err error
	if rc.SessionInfo.Expired() {
		rc.SessionInfo, err = ensureValidSessionInfo(cfg.APIServer, cfg.Platform, rc.Serial, ctx)
		if err != nil {
			return fmt.Errorf("ensuring valid session key: %v", err)
		}
	}
	return nil
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
			ID: os.Getpid() & 0xffff, Seq: 1, // << uint(seq), // TODO
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
