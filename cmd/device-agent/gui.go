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
		Version      *systray.MenuItem
		GatewayItems []*systray.MenuItem
	}
}

const (
	VersionClicked GuiEvent = iota
	StateInfoClicked
	ConnectClicked
	QuitClicked

	maxGateways         = 20
	slackURL            = "slack://channel?team=T5LNAMWNA&id=D011T20LDHD"
	softwareReleasePage = "https://github.com/nais/device/releases/latest"
)

func NewGUI() *Gui {
	gui := &Gui{}
	systray.SetIcon(readIcon("blue"))

	gui.MenuItems.Version = systray.AddMenuItem("Update to latest version...", "Click to open browser")
	gui.MenuItems.Version.Hide()
	systray.AddSeparator()
	gui.MenuItems.State = systray.AddMenuItem("", "State")
	gui.MenuItems.StateInfo = systray.AddMenuItem("", "StateExtra")
	gui.MenuItems.StateInfo.Hide()
	gui.MenuItems.State.Disable()
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
	gui.MenuItems.Quit = systray.AddMenuItem("Quit", "exit the application")

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
		systray.SetIcon(readIcon("red"))
		gui.MenuItems.Connect.Enable()
	case StateConnected:
		gui.MenuItems.Connect.SetTitle("Disconnect")
		systray.SetIcon(readIcon("green"))
		gui.MenuItems.Connect.Enable()
	case StateUnhealthy:
		gui.MenuItems.StateInfo.SetTitle("slack: /msg @kolide status")
		gui.MenuItems.StateInfo.Show()
		systray.SetIcon(readIcon("yellow"))
	default:
		gui.MenuItems.Connect.Disable()
	}
}

func (gui *Gui) handleNewVersion() {
	gui.MenuItems.Version.Show()
}
