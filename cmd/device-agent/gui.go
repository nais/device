package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/getlantern/systray"
	"github.com/nais/device/device-agent/apiserver"
)

type GuiEvent int

type Gui struct {
	ProgramState        chan ProgramState
	Gateways            chan apiserver.Gateways
	Events              chan GuiEvent
	Interrupts          chan os.Signal
	NewVersionAvailable chan bool
	MenuItems           struct {
		Connect      *systray.MenuItem
		Quit         *systray.MenuItem
		State        *systray.MenuItem
		StateInfo    *systray.MenuItem
		Logs         *systray.MenuItem
		DeviceLog    *systray.MenuItem
		HelperLog    *systray.MenuItem
		Version      *systray.MenuItem
		GatewayItems []*systray.MenuItem
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

func NewGUI() *Gui {
	gui := &Gui{}
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
	gui.MenuItems.GatewayItems = make([]*systray.MenuItem, maxGateways)

	for i := range gui.MenuItems.GatewayItems {
		gui.MenuItems.GatewayItems[i] = systray.AddMenuItem("", "")
		gui.MenuItems.GatewayItems[i].Disable()
		gui.MenuItems.GatewayItems[i].Hide()
	}
	systray.AddSeparator()
	gui.MenuItems.Quit = systray.AddMenuItem("Quit", "Exit the application")

	gui.Interrupts = make(chan os.Signal, 1)
	signal.Notify(gui.Interrupts, os.Interrupt, syscall.SIGTERM)

	gui.ProgramState = make(chan ProgramState, 8)
	gui.Gateways = make(chan apiserver.Gateways, 8)
	gui.Events = make(chan GuiEvent, 8)
	gui.NewVersionAvailable = make(chan bool, 8)

	return gui
}

func (gui *Gui) EventLoop() {
	for {
		select {
		case progstate := <-gui.ProgramState:
			gui.handleProgramState(progstate)
		case gateways := <-gui.Gateways:
			gui.handleGateways(gateways)
		case <-gui.NewVersionAvailable:
			gui.handleNewVersion()
		case <-gui.MenuItems.Version.ClickedCh:
			gui.Events <- VersionClicked
		case <-gui.MenuItems.StateInfo.ClickedCh:
			gui.Events <- StateInfoClicked
		case <-gui.MenuItems.Connect.ClickedCh:
			gui.Events <- ConnectClicked
		case <-gui.MenuItems.Quit.ClickedCh:
			gui.Events <- QuitClicked
		case <-gui.Interrupts:
			gui.Events <- QuitClicked
		case <-gui.MenuItems.DeviceLog.ClickedCh:
			gui.Events <- DeviceLogClicked
		case <-gui.MenuItems.HelperLog.ClickedCh:
			gui.Events <- HelperLogClicked
		}
	}
}

func (gui *Gui) handleGateways(gateways apiserver.Gateways) {
	max := len(gateways)
	if max > maxGateways {
		panic("twenty wasn't enough, was it??????")
	}
	for i, gateway := range gateways {
		gui.MenuItems.GatewayItems[i].SetTitle(gateway.Name)
		gui.MenuItems.GatewayItems[i].SetTooltip(gateway.Endpoint)
		if gateway.Healthy {
			gui.MenuItems.GatewayItems[i].Check()
		} else {
			gui.MenuItems.GatewayItems[i].Uncheck()
		}
		gui.MenuItems.GatewayItems[i].Show()
	}
	for i := max; i < maxGateways; i++ {
		gui.MenuItems.GatewayItems[i].Hide()
	}
}

func (gui *Gui) handleProgramState(state ProgramState) {
	gui.MenuItems.State.SetTitle("Status: " + state.String())
	gui.MenuItems.StateInfo.Hide()
	switch state {
	case StateNewVersion:
		gui.MenuItems.Version.Show()
	case StateDisconnected:
		gui.MenuItems.Connect.SetTitle("Connect")
		systray.SetIcon(NaisLogoRed)
	case StateConnected:
		systray.SetIcon(NaisLogoGreen)
	case StateUnhealthy:
		gui.MenuItems.StateInfo.SetTitle("slack: /msg @kolide status")
		gui.MenuItems.StateInfo.Show()
		systray.SetIcon(NaisLogoYellow)
	}

	if state != StateDisconnected {
		gui.MenuItems.Connect.SetTitle("Disconnect")
	}
}

func (gui *Gui) handleNewVersion() {
	gui.MenuItems.Version.Show()
}
