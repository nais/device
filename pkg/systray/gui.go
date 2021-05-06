package systray

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/nais/device/device-agent/open"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"

	"github.com/getlantern/systray"
)

type GuiEvent int

type GatewayItem struct {
	Gateway  *pb.Gateway
	MenuItem *systray.MenuItem
}

type Gui struct {
	DeviceAgentClient        pb.DeviceAgentClient
	AgentStatusChannel       chan *pb.AgentStatus
	AgentStatus              *pb.AgentStatus
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
		Settings     *systray.MenuItem
		AutoConnect *systray.MenuItem
		BlackAndWhite *systray.MenuItem
		DeviceLog    *systray.MenuItem
		HelperLog    *systray.MenuItem
		SystrayLog   *systray.MenuItem
		Version      *systray.MenuItem
		Upgrade      *systray.MenuItem
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
	LogClicked
	AutoConnectClicked
	BlackAndWhiteClicked

	maxGateways         = 20
	slackURL            = "slack://channel?team=T5LNAMWNA&id=D011T20LDHD"
	softwareReleasePage = "https://doc.nais.io/device/update/"
	requestBackoff      = 5 * time.Second
)

func NewGUI(client pb.DeviceAgentClient) *Gui {
	gui := &Gui{
		DeviceAgentClient: client,
	}
	applyDisconnectedIcon()

	gui.MenuItems.Version = systray.AddMenuItem("naisdevice "+version.Version, "")
	gui.MenuItems.Version.Disable()
	gui.MenuItems.Upgrade = systray.AddMenuItem("Update to latest version...", "Click to open browser")
	gui.MenuItems.Upgrade.Hide()
	systray.AddSeparator()
	gui.MenuItems.State = systray.AddMenuItem("", "")
	gui.MenuItems.StateInfo = systray.AddMenuItem("", "")
	gui.MenuItems.StateInfo.Hide()
	gui.MenuItems.State.Disable()
	gui.MenuItems.Logs = systray.AddMenuItem("Logs", "")
	gui.MenuItems.Settings = systray.AddMenuItem("Settings", "")
	gui.MenuItems.AutoConnect = gui.MenuItems.Settings.AddSubMenuItemCheckbox("Connect automatically on startup", "", cfg.AutoConnect)
	gui.MenuItems.BlackAndWhite = gui.MenuItems.Settings.AddSubMenuItemCheckbox("Black and white icons", "", cfg.BlackAndWhiteIcons)
	gui.MenuItems.DeviceLog = gui.MenuItems.Logs.AddSubMenuItem("Agent", "")
	gui.MenuItems.HelperLog = gui.MenuItems.Logs.AddSubMenuItem("Helper", "")
	gui.MenuItems.SystrayLog = gui.MenuItems.Logs.AddSubMenuItem("Systray", "")
	systray.AddSeparator()
	gui.MenuItems.Connect = systray.AddMenuItem("Connect", "")
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

	gui.AgentStatusChannel = make(chan *pb.AgentStatus, 8)
	gui.Events = make(chan GuiEvent, 8)
	gui.NewVersionAvailable = make(chan bool, 8)
	gui.PrivilegedGatewayClicked = make(chan string)

	return gui
}

func (gui *Gui) EventLoop() {
	for {
		select {
		case guiEvent := <-gui.Events:
			gui.handleGuiEvent(guiEvent)
			if guiEvent == QuitClicked {
				systray.Quit()
				return
			}
		case agentStatus := <-gui.AgentStatusChannel:
			gui.handleAgentStatus(agentStatus)
		case <-gui.NewVersionAvailable:
			gui.handleNewVersion()
		}
	}
}

func (gui *Gui) handleButtonClicks() {
	gui.aggregateGatewayButtonClicks()

	for {
		select {
		case <-gui.MenuItems.Upgrade.ClickedCh:
			gui.Events <- VersionClicked
		case <-gui.MenuItems.StateInfo.ClickedCh:
			gui.Events <- StateInfoClicked
		case <-gui.MenuItems.AutoConnect.ClickedCh:
			gui.Events <- AutoConnectClicked
		case <-gui.MenuItems.BlackAndWhite.ClickedCh:
			gui.Events <- BlackAndWhiteClicked
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
		case <-gui.MenuItems.SystrayLog.ClickedCh:
			gui.Events <- LogClicked
		case name := <-gui.PrivilegedGatewayClicked:
			accessPrivilegedGateway(name)
		}
	}
}

func (gui *Gui) handleAgentConnect() {
	gui.MenuItems.State.SetTitle("Refreshing Device Agent state...")
	gui.MenuItems.Connect.Enable()
}

func (gui *Gui) handleAgentDisconnect() {
	applyDisconnectedIcon()

	gui.MenuItems.State.SetTitle("Waiting for Device Agent...")
	gui.MenuItems.Connect.Disable()
	for i := range gui.MenuItems.GatewayItems {
		gui.MenuItems.GatewayItems[i].MenuItem.Disable()
		gui.MenuItems.GatewayItems[i].MenuItem.Hide()
	}
}

func (gui *Gui) handleAgentStatus(agentStatus *pb.AgentStatus) {
	log.Debugf("received agent status: %v", agentStatus)

	gui.AgentStatus = agentStatus

	switch agentStatus.GetConnectionState() {
	case pb.AgentState_Bootstrapping:
		gui.MenuItems.Connect.SetTitle("Disconnect")
	case pb.AgentState_Connected:
		gui.updateIcons()
	case pb.AgentState_Unhealthy:
		gui.updateIcons()
	case pb.AgentState_Disconnected:
		gui.updateIcons()
		gui.MenuItems.Connect.SetTitle("Connect")
	}

	gui.MenuItems.State.SetTitle(agentStatus.ConnectionStateString())
	if agentStatus.NewVersionAvailable {
		gui.MenuItems.Upgrade.Show()
	} else {
		gui.MenuItems.Upgrade.Hide()
	}

	gateways := agentStatus.GetGateways()
	sort.Slice(gateways, func(i, j int) bool {
		return gateways[i].GetName() < gateways[j].GetName()
	})

	max := len(gateways)
	if max > maxGateways {
		panic(fmt.Sprintf("cannot exceed %d gateways", maxGateways))
	}
	for i, gateway := range gateways {
		gui.MenuItems.GatewayItems[i].Gateway = gateway

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

func applyDisconnectedIcon() {
	if cfg.BlackAndWhiteIcons {
		systray.SetIcon(NaisLogoBwDisconnected)
	} else {
		systray.SetIcon(NaisLogoRed)
	}
}

func (gui* Gui) updateIcons() {
	if gui.AgentStatus.GetConnectionState() == pb.AgentState_Disconnected {
		applyDisconnectedIcon()
	} else if gui.AgentStatus.GetConnectionState() == pb.AgentState_Connected {
		if cfg.BlackAndWhiteIcons {
			systray.SetIcon(NaisLogoBwConnected)
		} else {
			systray.SetIcon(NaisLogoGreen)
		}
	} else if gui.AgentStatus.GetConnectionState() == pb.AgentState_Unhealthy {
		systray.SetIcon(NaisLogoYellow)
	}
}

func (gui* Gui) saveConfig() {
	err := WriteToJSONFile(cfg, cfg.ConfigDir + ConfigFile)
	if err != nil {
		log.Errorf("Error writing config to disk: %w", err)
	}
}

func (gui *Gui) handleGuiEvent(guiEvent GuiEvent) {
	switch guiEvent {
	case VersionClicked:
		err := open.Open(softwareReleasePage)
		if err != nil {
			log.Warnf("opening latest release url: %v", err)
		}

	case StateInfoClicked:
		err := open.Open(slackURL)
		if err != nil {
			log.Warnf("opening slack: %v", err)
		}

	case AutoConnectClicked:
		if cfg.AutoConnect {
			gui.MenuItems.AutoConnect.Uncheck()
		} else {
			gui.MenuItems.AutoConnect.Check()
		}
		cfg.AutoConnect = !cfg.AutoConnect
		gui.saveConfig()

	case BlackAndWhiteClicked:
		if cfg.BlackAndWhiteIcons {
			gui.MenuItems.BlackAndWhite.Uncheck()
		} else {
			gui.MenuItems.BlackAndWhite.Check()
		}
		cfg.BlackAndWhiteIcons = !cfg.BlackAndWhiteIcons
		gui.updateIcons()
		gui.saveConfig()

	case ConnectClicked:
		log.Infof("Connect button clicked")
		if gui.AgentStatus.GetConnectionState() == pb.AgentState_Disconnected {
			_, err := gui.DeviceAgentClient.Login(context.Background(), &pb.LoginRequest{})
			if err != nil {
				log.Errorf("connect: %v", err)
			}
		} else {
			_, err := gui.DeviceAgentClient.Logout(context.Background(), &pb.LogoutRequest{})
			if err != nil {
				log.Errorf("while disconnecting: %v", err)
			}
		}

	case HelperLogClicked:
		err := open.Open(filepath.Join(cfg.ConfigDir, "logs", "helper.log"))
		if err != nil {
			log.Warn("opening device agent helper log: %w", err)
		}

	case DeviceLogClicked:
		err := open.Open(filepath.Join(cfg.ConfigDir, "logs", "agent.log"))
		if err != nil {
			log.Warn("opening device agent log: %w", err)
		}

	case LogClicked:
		err := open.Open(filepath.Join(cfg.ConfigDir, "logs", "systray.log"))
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

func (gui *Gui) handleStatusStream() {
	for {
		gui.handleAgentDisconnect()

		ctx := context.Background()

		log.Infof("Requesting status updates from naisdevice-agent...")

		statusStream, err := gui.DeviceAgentClient.Status(ctx, &pb.AgentStatusRequest{})
		if err != nil {
			log.Errorf("Request status stream: %s", err)
			time.Sleep(requestBackoff)
			continue
		}

		log.Infof("naisdevice-agent status stream established")
		gui.handleAgentConnect()

		for {
			status, err := statusStream.Recv()
			if err != nil {
				log.Errorf("Receive status from device-agent stream: %v", err)
				break
			}

			gui.AgentStatusChannel <- status
		}

	}
}

func (gui *Gui) handleNewVersion() {
	gui.MenuItems.Upgrade.Show()
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
