package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nais/device/device-agent/open"
	"github.com/nais/device/pkg/logger"
	pb "github.com/nais/device/pkg/protobuf"
	log "github.com/sirupsen/logrus"

	"github.com/getlantern/systray"
	"github.com/nais/device/device-agent/apiserver"
)

type GuiEvent int

type GatewayItem struct {
	Gateway  apiserver.Gateway
	MenuItem *systray.MenuItem
}

type GuiState int

const (
	GuiStateConnected GuiState = iota
	GuiStateDisconnected
	GuiStateAwaitingAgent
	GuiStateConnecting
	GuiStateError
)

type Gui struct {
	DeviceAgentClient        pb.DeviceAgentClient
	Gateways                 chan apiserver.Gateways
	Events                   chan GuiEvent
	Interrupts               chan os.Signal
	NewVersionAvailable      chan bool
	PrivilegedGatewayClicked chan string
	MenuItems                struct {
		Connect      *systray.MenuItem
		Quit         *systray.MenuItem
		State        *systray.MenuItem
		StateInfo    *systray.MenuItem
		Logs         *systray.MenuItem
		DeviceLog    *systray.MenuItem
		HelperLog    *systray.MenuItem
		Version      *systray.MenuItem
		GatewayItems []*GatewayItem
	}
}

const (
	VersionClicked GuiEvent = iota
	StateInfoClicked
	ConnectClicked
	QuitClicked
	DeviceLogClicked
	HelperLogClicked

	maxGateways         = 20
	slackURL            = "slack://channel?team=T5LNAMWNA&id=D011T20LDHD"
	softwareReleasePage = "https://doc.nais.io/device/install"
)

func NewGUI(client pb.DeviceAgentClient) *Gui {
	gui := &Gui{
		DeviceAgentClient: client,
	}
	systray.SetIcon(NaisLogoBlue)

	gui.MenuItems.Version = systray.AddMenuItem("Update to latest version...", "Click to open browser")
	gui.MenuItems.Version.Hide()
	systray.AddSeparator()
	gui.MenuItems.State = systray.AddMenuItem("", "State")
	gui.MenuItems.StateInfo = systray.AddMenuItem("", "StateExtra")
	gui.MenuItems.StateInfo.Hide()
	gui.MenuItems.State.Disable()
	gui.MenuItems.Logs = systray.AddMenuItem("Logs", "")
	gui.MenuItems.DeviceLog = gui.MenuItems.Logs.AddSubMenuItem("Device Agent", "")
	gui.MenuItems.HelperLog = gui.MenuItems.Logs.AddSubMenuItem("Device Agent helper", "")
	systray.AddSeparator()
	gui.MenuItems.Connect = systray.AddMenuItem("Connect", "Bootstrap the nais device")
	systray.AddSeparator()
	gui.MenuItems.GatewayItems = make([]*GatewayItem, maxGateways)

	for i := range gui.MenuItems.GatewayItems {
		gui.MenuItems.GatewayItems[i] = &GatewayItem{}
		gui.MenuItems.GatewayItems[i].MenuItem = systray.AddMenuItem("", "")
		gui.MenuItems.GatewayItems[i].MenuItem.Disable()
		gui.MenuItems.GatewayItems[i].MenuItem.Hide()
	}

	systray.AddSeparator()
	gui.MenuItems.Quit = systray.AddMenuItem("Quit", "Exit the application")

	gui.Interrupts = make(chan os.Signal, 1)
	signal.Notify(gui.Interrupts, os.Interrupt, syscall.SIGTERM)

	gui.Gateways = make(chan apiserver.Gateways, 8)
	gui.Events = make(chan GuiEvent, 8)
	gui.NewVersionAvailable = make(chan bool, 8)
	gui.PrivilegedGatewayClicked = make(chan string)

	return gui
}

func (gui *Gui) EventLoop() {
	currentState := GuiStateDisconnected

	for {
		select {
		case guiEvent := <-gui.Events:
			gui.handleGuiEvent(guiEvent, currentState)
			if guiEvent == QuitClicked {
				systray.Quit()
				return
			}
		case gateways := <-gui.Gateways:
			gui.handleGateways(gateways)
		case <-gui.NewVersionAvailable:
			gui.handleNewVersion()
		}
	}
}

func (gui *Gui) handleButtonClicks() {
	gui.aggregateGatewayButtonClicks()

	for {
		select {
		case <-gui.MenuItems.Version.ClickedCh:
			gui.Events <- VersionClicked
		case <-gui.MenuItems.StateInfo.ClickedCh:
			gui.Events <- StateInfoClicked
		case <-gui.MenuItems.Connect.ClickedCh:
			gui.Events <- ConnectClicked
		case <-gui.MenuItems.Quit.ClickedCh:
			gui.Events <- QuitClicked
			return
		case <-gui.Interrupts:
			gui.Events <- QuitClicked
			return
		case <-gui.MenuItems.DeviceLog.ClickedCh:
			gui.Events <- DeviceLogClicked
		case <-gui.MenuItems.HelperLog.ClickedCh:
			gui.Events <- HelperLogClicked
		case name := <-gui.PrivilegedGatewayClicked:
			accessPrivilegedGateway(name)
		}
	}
}

func (gui *Gui) handleGateways(gateways apiserver.Gateways) {
	max := len(gateways)
	if max > maxGateways {
		panic("twenty wasn't enough, was it??????")
	}
	for i, gateway := range gateways {
		gui.MenuItems.GatewayItems[i].Gateway = *gateway

		menuItem := gui.MenuItems.GatewayItems[i].MenuItem
		menuItem.SetTitle(gateway.Name)
		menuItem.SetTooltip(gateway.Endpoint)

		if gateway.Healthy {
			menuItem.Check()
		} else {
			menuItem.Uncheck()
		}
		menuItem.Show()

		if gateway.RequiresPrivilegedAccess {
			menuItem.Enable()
		} else {
			menuItem.Disable()
		}
	}
	for i := max; i < maxGateways; i++ {
		gui.MenuItems.GatewayItems[i].MenuItem.Hide()
	}
}

func (gui *Gui) handleGuiEvent(guiEvent GuiEvent, state GuiState) {
	switch guiEvent {
	case VersionClicked:
		err := open.Open(softwareReleasePage)
		if err != nil {
			log.Warn("opening latest release url: %w", err)
		}

	case StateInfoClicked:
		err := open.Open(slackURL)
		if err != nil {
			log.Warnf("opening slack: %v", err)
		}

	case ConnectClicked:
		log.Infof("Connect button clicked")
		if state == GuiStateDisconnected {
			_, err := gui.DeviceAgentClient.Login(context.Background(), &pb.LoginRequest{})
			if err != nil {
				log.Fatalf("while connecting: %v", err)
			}
		} else {
			_, err := gui.DeviceAgentClient.Logout(context.Background(), &pb.LogoutRequest{})
			if err != nil {
				log.Fatalf("while disconnecting: %v", err)
			}
		}

	case HelperLogClicked:
		err := open.Open(logger.DeviceAgentHelperLogFilePath())
		if err != nil {
			log.Warn("opening device agent helper log: %w", err)
		}

	case DeviceLogClicked:
		err := open.Open(logger.DeviceAgentLogFilePath(cfg.configDir))
		if err != nil {
			log.Warn("opening device agent log: %w", err)
		}

	case QuitClicked:
		_, err := gui.DeviceAgentClient.Logout(context.Background(), &pb.LogoutRequest{})
		if err != nil {
			log.Fatalf("while disconnecting: %v", err)
		}
	}
}

func (gui *Gui) handleNewVersion() {
	gui.MenuItems.Version.Show()
}

func (gui *Gui) aggregateGatewayButtonClicks() {
	// Start a forwarder for each buttons click-channel and aggregates to a single channel
	for _, gatewayItem := range gui.MenuItems.GatewayItems {
		go func(gw *GatewayItem) {
			for range gw.MenuItem.ClickedCh {
				gui.PrivilegedGatewayClicked <- gw.Gateway.Name
			}
		}(gatewayItem)
	}
}

func accessPrivilegedGateway(gatewayName string) {
	err := open.Open(fmt.Sprintf("https://naisdevice-jita.nais.io/?gateway=%s", gatewayName))
	if err != nil {
		log.Errorf("opening browser: %v", err)
		// TODO: show error in gui (systray)
	}
}
