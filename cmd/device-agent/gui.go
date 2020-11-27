package main

import (
	"fmt"
	"github.com/nais/device/device-agent/open"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"

	"github.com/getlantern/systray"
	"github.com/nais/device/device-agent/apiserver"
)

type GuiEvent int

type Gui struct {
	ProgramState             chan ProgramState
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
	var previousGateways apiserver.Gateways
	go gui.anyGatewayClicked(&previousGateways)
	for {
		select {
		case progstate := <-gui.ProgramState:
			gui.handleProgramState(progstate)
		case gateways := <-gui.Gateways:
			previousGateways = gateways
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
		case name := <-gui.PrivilegedGatewayClicked:
			err := open.Open(fmt.Sprintf("https://naisdevice-jita.prod-gcp.nais.io/?gateway=%s", name))
			if err != nil {
				log.Errorf("opening browser: %v", err)
			}
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
		gui.MenuItems.GatewayItems[i].Enable()
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

// stop scrolling here
func (gui *Gui) anyGatewayClicked(gateways *apiserver.Gateways) {
	for {
		select {
		case <-gui.MenuItems.GatewayItems[0].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[0].Name
		case <-gui.MenuItems.GatewayItems[1].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[1].Name
		case <-gui.MenuItems.GatewayItems[2].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[2].Name
		case <-gui.MenuItems.GatewayItems[3].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[3].Name
		case <-gui.MenuItems.GatewayItems[4].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[4].Name
		case <-gui.MenuItems.GatewayItems[5].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[5].Name
		case <-gui.MenuItems.GatewayItems[6].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[6].Name
		case <-gui.MenuItems.GatewayItems[7].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[7].Name
		case <-gui.MenuItems.GatewayItems[8].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[8].Name
		case <-gui.MenuItems.GatewayItems[9].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[9].Name
		case <-gui.MenuItems.GatewayItems[10].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[10].Name
		case <-gui.MenuItems.GatewayItems[11].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[11].Name
		case <-gui.MenuItems.GatewayItems[12].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[12].Name
		case <-gui.MenuItems.GatewayItems[13].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[13].Name
		case <-gui.MenuItems.GatewayItems[14].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[14].Name
		case <-gui.MenuItems.GatewayItems[15].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[15].Name
		case <-gui.MenuItems.GatewayItems[16].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[16].Name
		case <-gui.MenuItems.GatewayItems[17].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[17].Name
		case <-gui.MenuItems.GatewayItems[18].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[18].Name
		case <-gui.MenuItems.GatewayItems[19].ClickedCh:
			gui.PrivilegedGatewayClicked <- (*gateways)[19].Name
		}
	}
}
