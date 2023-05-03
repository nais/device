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

	"github.com/nais/device/pkg/helper"

	"github.com/nais/device/assets"
	"github.com/nais/device/pkg/device-agent/open"
	"github.com/nais/device/pkg/notify"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/version"

	"github.com/getlantern/systray"
	log "github.com/sirupsen/logrus"
)

type GuiEvent int

type GatewayItem struct {
	Gateway  *pb.Gateway
	MenuItem *systray.MenuItem
}

type TenantItem struct {
	Tenant   *pb.Tenant
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
	TenantItemClicked        chan string
	ProgramContext           context.Context
	MenuItems                struct {
		Connect       *systray.MenuItem
		Quit          *systray.MenuItem
		State         *systray.MenuItem
		StateInfo     *systray.MenuItem
		Logs          *systray.MenuItem
		Settings      *systray.MenuItem
		AutoConnect   *systray.MenuItem
		BlackAndWhite *systray.MenuItem
		DeviceLog     *systray.MenuItem
		HelperLog     *systray.MenuItem
		SystrayLog    *systray.MenuItem
		ZipLog        *systray.MenuItem
		Version       *systray.MenuItem
		Upgrade       *systray.MenuItem
		Tenant        *systray.MenuItem
		TenantItems   []*TenantItem
		GatewayItems  []*GatewayItem
		AcceptableUse *systray.MenuItem
	}
	Config Config
}

const (
	VersionClicked GuiEvent = iota
	StateInfoClicked
	ConnectClicked
	QuitClicked
	DeviceLogClicked
	HelperLogClicked
	ZipLogsClicked
	LogClicked
	AutoConnectClicked
	BlackAndWhiteClicked
	AcceptableUseClicked

	maxTenants          = 10
	maxGateways         = 30
	slackURL            = "slack://channel?team=T5LNAMWNA&id=D011T20LDHD"
	softwareReleasePage = "https://doc.nais.io/device/update/"
	requestBackoff      = 5 * time.Second
)

func NewGUI(ctx context.Context, client pb.DeviceAgentClient, cfg Config) *Gui {
	gui := &Gui{
		DeviceAgentClient: client,
		Config:            cfg,
		ProgramContext:    ctx,
	}
	gui.applyDisconnectedIcon()

	gui.MenuItems.Version = systray.AddMenuItem("naisdevice "+version.Version, "")
	gui.MenuItems.Version.Disable()
	gui.MenuItems.Upgrade = systray.AddMenuItem("Update to latest version...", "Click to open browser")
	gui.MenuItems.Upgrade.Hide()
	systray.AddSeparator()
	gui.MenuItems.State = systray.AddMenuItem("", "")
	gui.MenuItems.StateInfo = systray.AddMenuItem("", "")
	gui.MenuItems.StateInfo.Hide()
	gui.MenuItems.State.Disable()
	gui.MenuItems.AcceptableUse = systray.AddMenuItem("Acceptable use policy", "")
	gui.MenuItems.AcceptableUse.Hide()
	gui.MenuItems.Logs = systray.AddMenuItem("Logs", "")
	gui.MenuItems.Settings = systray.AddMenuItem("Settings", "")
	gui.MenuItems.AutoConnect = gui.MenuItems.Settings.AddSubMenuItemCheckbox("Connect automatically on startup", "", false)
	gui.MenuItems.BlackAndWhite = gui.MenuItems.Settings.AddSubMenuItemCheckbox("Black and white icons", "", cfg.BlackAndWhiteIcons)
	gui.MenuItems.DeviceLog = gui.MenuItems.Logs.AddSubMenuItem("Agent", "")
	gui.MenuItems.HelperLog = gui.MenuItems.Logs.AddSubMenuItem("Helper", "")
	gui.MenuItems.SystrayLog = gui.MenuItems.Logs.AddSubMenuItem("Systray", "")
	gui.MenuItems.ZipLog = gui.MenuItems.Logs.AddSubMenuItem("Zip logfiles", "")
	gui.MenuItems.Tenant = systray.AddMenuItem("Tenant", "")
	gui.MenuItems.Tenant.Hide()
	systray.AddSeparator()
	gui.MenuItems.Connect = systray.AddMenuItem("Connect", "")
	systray.AddSeparator()
	gui.MenuItems.GatewayItems = make([]*GatewayItem, maxGateways)
	gui.MenuItems.TenantItems = make([]*TenantItem, maxTenants)

	for i := range gui.MenuItems.GatewayItems {
		gui.MenuItems.GatewayItems[i] = &GatewayItem{}
		gui.MenuItems.GatewayItems[i].MenuItem = systray.AddMenuItemCheckbox("", "", false)
		gui.MenuItems.GatewayItems[i].MenuItem.Disable()
		gui.MenuItems.GatewayItems[i].MenuItem.Hide()
	}

	for i := range gui.MenuItems.TenantItems {
		gui.MenuItems.TenantItems[i] = &TenantItem{}
		gui.MenuItems.TenantItems[i].MenuItem = gui.MenuItems.Tenant.AddSubMenuItemCheckbox("", "", false)
		gui.MenuItems.TenantItems[i].MenuItem.Disable()
		gui.MenuItems.TenantItems[i].MenuItem.Hide()
	}

	systray.AddSeparator()
	gui.MenuItems.Quit = systray.AddMenuItem("Quit", "Exit the application")

	gui.Interrupts = make(chan os.Signal, 1)
	signal.Notify(gui.Interrupts, os.Interrupt, syscall.SIGTERM)

	gui.AgentStatusChannel = make(chan *pb.AgentStatus, 8)
	gui.Events = make(chan GuiEvent, 8)
	gui.NewVersionAvailable = make(chan bool, 8)
	gui.PrivilegedGatewayClicked = make(chan string)
	gui.TenantItemClicked = make(chan string)

	return gui
}

func (gui *Gui) EventLoop(ctx context.Context) {
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
		case <-ctx.Done():
			systray.Quit()
			return
		}
	}
}

func (gui *Gui) handleButtonClicks(ctx context.Context) {
	gui.aggregateGatewayButtonClicks()
	gui.aggregateTenantButtonClicks()

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
		case <-gui.MenuItems.ZipLog.ClickedCh:
			gui.Events <- ZipLogsClicked
			return
		case name := <-gui.PrivilegedGatewayClicked:
			accessPrivilegedGateway(name)
		case name := <-gui.TenantItemClicked:
			gui.activateTenant(ctx, name)
		case <-gui.MenuItems.AcceptableUse.ClickedCh:
			gui.Events <- AcceptableUseClicked
		case <-ctx.Done():
			return
		}
	}
}

func (gui *Gui) updateGuiAgentConfig(config *pb.AgentConfiguration) {
	gui.MenuItems.AutoConnect.Enable()
	if config.AutoConnect {
		gui.MenuItems.AutoConnect.Check()
	}
}

func (gui *Gui) resetGuiAgentConfig() {
	gui.MenuItems.AutoConnect.Disable()
	gui.MenuItems.AutoConnect.Uncheck()
}

func (gui *Gui) handleAgentConnect() {
	gui.MenuItems.State.SetTitle("Refreshing Device Agent state...")

	response, err := gui.DeviceAgentClient.GetAgentConfiguration(gui.ProgramContext, &pb.GetAgentConfigurationRequest{})
	if err != nil {
		notify.Errorf("Failed to get initial agent config: %v", err)
	}
	gui.updateGuiAgentConfig(response.Config)
	gui.MenuItems.Connect.Enable()
}

func (gui *Gui) handleAgentDisconnect() {
	gui.MenuItems.State.SetTitle("Waiting for Device Agent...")

	gui.applyDisconnectedIcon()
	gui.resetGuiAgentConfig()
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

	tenants := agentStatus.GetTenants()
	if len(tenants) <= 1 {
		if len(tenants) == 1 && (tenants[0].Name == "NAV" || tenants[0].Name == "nav.no") {
			gui.MenuItems.AcceptableUse.Show()
		}
		return
	}

	gui.MenuItems.Tenant.Show()
	sort.Slice(tenants, func(i, j int) bool {
		return tenants[i].GetName() < tenants[j].GetName()
	})

	gui.MenuItems.AcceptableUse.Hide()
	for i, tenant := range tenants {
		gui.MenuItems.TenantItems[i].Tenant = tenant

		menuItem := gui.MenuItems.TenantItems[i].MenuItem
		menuItem.SetTitle(tenant.Name)
		menuItem.Show()
		menuItem.Enable()
		if tenant.Active {
			if tenant.Name == "NAV" || tenant.Name == "nav.no" {
				gui.MenuItems.AcceptableUse.Show()
			}
			menuItem.Check()
		} else {
			menuItem.Uncheck()
		}
	}
}

func (gui *Gui) applyDisconnectedIcon() {
	if gui.Config.BlackAndWhiteIcons {
		systray.SetTemplateIcon(assets.NaisLogoBwDisconnected, assets.NaisLogoBwDisconnected)
	} else {
		systray.SetIcon(assets.NaisLogoRed)
	}
}

func (gui *Gui) updateIcons() {
	if gui.AgentStatus.GetConnectionState() == pb.AgentState_Disconnected {
		gui.applyDisconnectedIcon()
	} else if gui.AgentStatus.GetConnectionState() == pb.AgentState_Connected {
		if gui.Config.BlackAndWhiteIcons {
			systray.SetTemplateIcon(assets.NaisLogoBwConnected, assets.NaisLogoBwConnected)
		} else {
			systray.SetIcon(assets.NaisLogoGreen)
		}
	} else if gui.AgentStatus.GetConnectionState() == pb.AgentState_Unhealthy {
		systray.SetIcon(assets.NaisLogoYellow)
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
		getConfigResponse, err := gui.DeviceAgentClient.GetAgentConfiguration(context.Background(), &pb.GetAgentConfigurationRequest{})
		if err != nil {
			log.Errorf("get agent config: %v", err)
			break
		}

		getConfigResponse.Config.AutoConnect = !gui.MenuItems.AutoConnect.Checked()
		setConfigRequest := &pb.SetAgentConfigurationRequest{Config: getConfigResponse.Config}

		_, err = gui.DeviceAgentClient.SetAgentConfiguration(context.Background(), setConfigRequest)
		if err != nil {
			log.Errorf("set agent config: %v", err)
			break
		}

		if gui.MenuItems.AutoConnect.Checked() {
			gui.MenuItems.AutoConnect.Uncheck()
		} else {
			gui.MenuItems.AutoConnect.Check()
		}

	case BlackAndWhiteClicked:
		if gui.Config.BlackAndWhiteIcons {
			gui.MenuItems.BlackAndWhite.Uncheck()
		} else {
			gui.MenuItems.BlackAndWhite.Check()
		}
		gui.Config.BlackAndWhiteIcons = !gui.Config.BlackAndWhiteIcons
		gui.updateIcons()
		gui.Config.Persist()

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
		err := open.Open(filepath.Join(gui.Config.ConfigDir, "logs", "helper.log"))
		if err != nil {
			log.Warn("opening device agent helper log: %w", err)
		}

	case DeviceLogClicked:
		err := open.Open(filepath.Join(gui.Config.ConfigDir, "logs", "agent.log"))
		if err != nil {
			log.Warn("opening device agent log: %w", err)
		}

	case ZipLogsClicked:
		logDir := filepath.Join(gui.Config.ConfigDir, "logs")
		logFiles := [3]string{
			filepath.Join(logDir, "helper.log"),
			filepath.Join(logDir, "agent.log"),
			filepath.Join(logDir, "systray.log"),
		}
		zipLocation, err := helper.ZipLogFiles(logFiles[:])
		if err != nil {
			log.Errorf("zipping log files: %v", err)
		}
		err = open.Open("file://" + filepath.Dir(zipLocation))
		if err != nil {
			log.Errorf("open %v", err)
		}

	case LogClicked:
		err := open.Open(filepath.Join(gui.Config.ConfigDir, "logs", "systray.log"))
		if err != nil {
			log.Warn("opening device agent log: %w", err)
		}

	case AcceptableUseClicked:
		err := open.Open("https://naisdevice-approval.nais.io/")
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

func (gui *Gui) handleStatusStream(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Infof("stopping handleStatusStream as context is done")
			return
		default:
			gui.handleAgentDisconnect()
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

func (gui *Gui) aggregateTenantButtonClicks() {
	// Start a forwarder for each buttons click-channel and aggregates to a single channel
	for _, tenantItem := range gui.MenuItems.TenantItems {
		go func(item *TenantItem) {
			for range item.MenuItem.ClickedCh {
				gui.TenantItemClicked <- item.Tenant.Name
			}
		}(tenantItem)
	}
}

func accessPrivilegedGateway(gatewayName string) {
	err := open.Open(fmt.Sprintf("https://naisdevice-jita.nais.io/?gateway=%s", gatewayName))
	if err != nil {
		log.Errorf("opening browser: %v", err)
		// TODO: show error in gui (systray)
	}
}

func (gui *Gui) activateTenant(ctx context.Context, name string) {
	req := &pb.SetActiveTenantRequest{
		Name: name,
	}
	_, err := gui.DeviceAgentClient.SetActiveTenant(ctx, req)
	if err != nil {
		notify.Errorf("Failed to activate tenant, err: %v", err)
		return
	}

	getConfigResponse, err := gui.DeviceAgentClient.GetAgentConfiguration(ctx, &pb.GetAgentConfigurationRequest{})
	if err != nil {
		log.Errorf("Failed to get agent configuration, err: %v", err)
		return
	}

	if getConfigResponse.Config.AutoConnect {
		_, err = gui.DeviceAgentClient.Login(ctx, &pb.LoginRequest{})
		if err != nil {
			log.Errorf("connect: %v", err)
		}
	}
}
