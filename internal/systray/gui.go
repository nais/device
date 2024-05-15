package systray

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/nais/device/internal/helper"
	helperconfig "github.com/nais/device/internal/helper/config"
	"github.com/nais/device/internal/logger"
	"github.com/nais/device/internal/otel"

	"github.com/nais/device/assets"
	"github.com/nais/device/internal/device-agent/open"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/version"

	"fyne.io/systray"
	"github.com/sirupsen/logrus"
)

type GuiEvent int

type GatewayItem struct {
	Gateway  *pb.Gateway
	MenuItem *cachedMenuItem
}

type TenantItem struct {
	Tenant   *pb.Tenant
	MenuItem *cachedMenuItem
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
		Connect       *cachedMenuItem
		Quit          *cachedMenuItem
		State         *cachedMenuItem
		StateInfo     *cachedMenuItem
		Logs          *cachedMenuItem
		Settings      *cachedMenuItem
		AutoConnect   *cachedMenuItem
		BlackAndWhite *cachedMenuItem
		DeviceLog     *cachedMenuItem
		HelperLog     *cachedMenuItem
		SystrayLog    *cachedMenuItem
		ZipLog        *cachedMenuItem
		Version       *cachedMenuItem
		Upgrade       *cachedMenuItem
		Tenant        *cachedMenuItem
		TenantItems   []*TenantItem
		GatewayItems  []*GatewayItem
		AcceptableUse *cachedMenuItem
	}
	Config   Config
	notifier notify.Notifier
	log      *logrus.Entry

	icon         []byte
	templateIcon []byte
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
	softwareReleasePage = "https://doc.nais.io/operate/naisdevice/how-to/update/"
	requestBackoff      = 5 * time.Second
)

func NewGUI(ctx context.Context, log *logrus.Entry, client pb.DeviceAgentClient, cfg Config, notifier notify.Notifier) *Gui {
	gui := &Gui{
		DeviceAgentClient: client,
		Config:            cfg,
		ProgramContext:    ctx,
		notifier:          notifier,
		log:               log,
	}
	gui.applyDisconnectedIcon()

	gui.MenuItems.Version = AddMenuItem("naisdevice "+version.Version, "")
	gui.MenuItems.Version.Disable()
	gui.MenuItems.Upgrade = AddMenuItem("Update to latest version...", "Click to open browser")
	gui.MenuItems.Upgrade.Hide()
	systray.AddSeparator()
	gui.MenuItems.State = AddMenuItem("", "")
	gui.MenuItems.StateInfo = AddMenuItem("", "")
	gui.MenuItems.StateInfo.Hide()
	gui.MenuItems.State.Disable()
	gui.MenuItems.AcceptableUse = AddMenuItem("Acceptable use policy", "")
	gui.MenuItems.AcceptableUse.Hide()
	gui.MenuItems.Logs = AddMenuItem("Logs", "")
	gui.MenuItems.Settings = AddMenuItem("Settings", "")
	gui.MenuItems.AutoConnect = gui.MenuItems.Settings.AddSubMenuItemCheckbox("Connect automatically on startup", "", false)
	gui.MenuItems.BlackAndWhite = gui.MenuItems.Settings.AddSubMenuItemCheckbox("Black and white icons", "", cfg.BlackAndWhiteIcons)
	gui.MenuItems.DeviceLog = gui.MenuItems.Logs.AddSubMenuItem("Agent", "")
	gui.MenuItems.HelperLog = gui.MenuItems.Logs.AddSubMenuItem("Helper", "")
	gui.MenuItems.SystrayLog = gui.MenuItems.Logs.AddSubMenuItem("Systray", "")
	gui.MenuItems.ZipLog = gui.MenuItems.Logs.AddSubMenuItem("Zip logfiles", "")
	gui.MenuItems.Tenant = AddMenuItem("Tenant", "")
	gui.MenuItems.Tenant.Hide()
	systray.AddSeparator()
	gui.MenuItems.Connect = AddMenuItem("Connect", "")
	systray.AddSeparator()
	gui.MenuItems.GatewayItems = make([]*GatewayItem, maxGateways)
	gui.MenuItems.TenantItems = make([]*TenantItem, maxTenants)

	for i := range gui.MenuItems.GatewayItems {
		gui.MenuItems.GatewayItems[i] = &GatewayItem{}
		gui.MenuItems.GatewayItems[i].MenuItem = AddMenuItemCheckbox("", "", false)
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
	gui.MenuItems.Quit = AddMenuItem("Quit", "Exit the application")

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
			ctx, span := otel.Start(ctx, "event/"+guiEvent.String())
			gui.handleGuiEvent(ctx, guiEvent)
			span.End()
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
		gui.notifier.Errorf("Failed to get initial agent config: %v", err)
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
	gui.log.Debugf("received agent status: %v", agentStatus)

	gui.AgentStatus = agentStatus

	switch agentStatus.GetConnectionState() {
	case pb.AgentState_Authenticating:
		fallthrough
	case pb.AgentState_Bootstrapping:
		fallthrough
	case pb.AgentState_Connected:
		fallthrough
	case pb.AgentState_Unhealthy:
		gui.MenuItems.Connect.SetTitle("Disconnect")
		gui.updateIcons()
	case pb.AgentState_Disconnected:
		gui.MenuItems.Connect.SetTitle("Connect")
		gui.updateIcons()
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

func (gui *Gui) setIcon(icon []byte) {
	if !bytes.Equal(gui.icon, icon) {
		gui.icon = icon
		systray.SetIcon(icon)
	}
}

func (gui *Gui) setTemplateIcon(templateIcon, icon []byte) {
	if !bytes.Equal(gui.templateIcon, templateIcon) || !bytes.Equal(gui.icon, icon) {
		gui.templateIcon = templateIcon
		gui.icon = icon
		systray.SetTemplateIcon(templateIcon, icon)
	}
}

func (gui *Gui) applyDisconnectedIcon() {
	if gui.Config.BlackAndWhiteIcons {
		gui.setTemplateIcon(assets.NaisLogoBwDisconnected, assets.NaisLogoBwDisconnected)
	} else {
		gui.setIcon(assets.NaisLogoRed)
	}
}

func (gui *Gui) updateIcons() {
	switch gui.AgentStatus.GetConnectionState() {
	case pb.AgentState_Connected:
		if gui.Config.BlackAndWhiteIcons {
			gui.setTemplateIcon(assets.NaisLogoBwConnected, assets.NaisLogoBwConnected)
		} else {
			gui.setIcon(assets.NaisLogoGreen)
		}
	case pb.AgentState_Disconnected:
		gui.applyDisconnectedIcon()
	case pb.AgentState_Unhealthy:
		if !bytes.Equal(gui.icon, assets.NaisLogoYellow) {
			gui.setIcon(assets.NaisLogoYellow)
		}
	}
}

func (gui *Gui) handleGuiEvent(ctx context.Context, guiEvent GuiEvent) {
	switch guiEvent {
	case VersionClicked:
		open.Open(softwareReleasePage)

	case StateInfoClicked:
		open.Open(slackURL)

	case AutoConnectClicked:
		getConfigResponse, err := gui.DeviceAgentClient.GetAgentConfiguration(ctx, &pb.GetAgentConfigurationRequest{})
		if err != nil {
			gui.log.Errorf("get agent config: %v", err)
			break
		}

		getConfigResponse.Config.AutoConnect = !gui.MenuItems.AutoConnect.Checked()
		setConfigRequest := &pb.SetAgentConfigurationRequest{Config: getConfigResponse.Config}

		_, err = gui.DeviceAgentClient.SetAgentConfiguration(ctx, setConfigRequest)
		if err != nil {
			gui.log.Errorf("set agent config: %v", err)
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
		gui.log.Infof("Connect button clicked")
		if gui.AgentStatus.GetConnectionState() == pb.AgentState_Disconnected {
			_, err := gui.DeviceAgentClient.Login(ctx, &pb.LoginRequest{})
			if err != nil {
				gui.log.Errorf("connect: %v", err)
			}
		} else {
			_, err := gui.DeviceAgentClient.Logout(ctx, &pb.LogoutRequest{})
			if err != nil {
				gui.log.Errorf("while disconnecting: %v", err)
			}
		}

	case HelperLogClicked:
		open.Open(logger.LatestFilepath(helperconfig.LogDir, logger.Helper))

	case DeviceLogClicked:
		logDirPath := filepath.Join(gui.Config.ConfigDir, logger.LogDir)
		open.Open(logger.LatestFilepath(logDirPath, logger.Agent))

	case ZipLogsClicked:
		userLogDirPath := filepath.Join(gui.Config.ConfigDir, logger.LogDir)
		logFiles := [3]string{
			logger.LatestFilepath(userLogDirPath, logger.Agent),
			logger.LatestFilepath(helperconfig.LogDir, logger.Helper),
			logger.LatestFilepath(userLogDirPath, logger.Systray),
		}
		zipLocation, err := helper.ZipLogFiles(logFiles[:])
		if err != nil {
			gui.log.Errorf("zipping log files: %v", err)
		}
		open.Open("file://" + filepath.Dir(zipLocation))

	case LogClicked:
		logDirPath := filepath.Join(gui.Config.ConfigDir, logger.LogDir)
		open.Open(logger.LatestFilepath(logDirPath, logger.Systray))

	case AcceptableUseClicked:
		open.Open("https://naisdevice-approval.external.prod-gcp.nav.cloud.nais.io/")
	case QuitClicked:
		_, err := gui.DeviceAgentClient.Logout(ctx, &pb.LogoutRequest{})
		if err != nil {
			gui.log.Fatalf("while disconnecting: %v", err)
		}
	}
}

func (gui *Gui) handleStatusStream(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			gui.log.Infof("stopping handleStatusStream as context is done")
			return
		default:
			gui.handleAgentDisconnect()
			gui.log.Infof("Requesting status updates from naisdevice-agent...")

			statusStream, err := gui.DeviceAgentClient.Status(ctx, &pb.AgentStatusRequest{})
			if err != nil {
				gui.log.Errorf("Request status stream: %s", err)
				time.Sleep(requestBackoff)
				continue
			}

			gui.log.Infof("naisdevice-agent status stream established")
			gui.handleAgentConnect()

			for {
				status, err := statusStream.Recv()
				if err != nil {
					gui.log.Errorf("Receive status from device-agent stream: %v", err)
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
	open.Open(fmt.Sprintf("https://naisdevice-jita.external.prod-gcp.nav.cloud.nais.io/?gateway=%s", gatewayName))
}

func (gui *Gui) activateTenant(ctx context.Context, name string) {
	req := &pb.SetActiveTenantRequest{
		Name: name,
	}
	_, err := gui.DeviceAgentClient.SetActiveTenant(ctx, req)
	if err != nil {
		gui.notifier.Errorf("Failed to activate tenant, err: %v", err)
		return
	}

	getConfigResponse, err := gui.DeviceAgentClient.GetAgentConfiguration(ctx, &pb.GetAgentConfigurationRequest{})
	if err != nil {
		gui.log.Errorf("Failed to get agent configuration, err: %v", err)
		return
	}

	if getConfigResponse.Config.AutoConnect {
		_, err = gui.DeviceAgentClient.Login(ctx, &pb.LoginRequest{})
		if err != nil {
			gui.log.Errorf("connect: %v", err)
		}
	}
}
