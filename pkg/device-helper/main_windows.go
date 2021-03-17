package device_helper

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"time"

	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
)

const (
	WireGuardBinary = `c:\Program Files\WireGuard\wireguard.exe`
	ServiceName     = "naisdevice-agent-helper"
)

type MyService struct {
	controlChannel chan ControlEvent
}

type ControlEvent int

type Controllable interface {
	ControlChannel() <-chan ControlEvent
}

type WindowsConfigurator struct {
	helperConfig       Config
	oldWireGuardConfig []byte
	wgNeedsRestart     bool
}

func New(helperConfig Config) *WindowsConfigurator {
	return &WindowsConfigurator{
		helperConfig: helperConfig,
	}
}

const (
	Stop ControlEvent = iota
	Pause
	Continue
)

func (configurator *WindowsConfigurator) Prerequisites() error {
	if err := filesExist(WireGuardBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
	}

	isWindowsService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Checking if session is windows service: %v", err)
	}

	if !isWindowsService {
		return nil
	}

	go func() {
		s := NewService()
		err = svc.Run(ServiceName, s)
		if err != nil {
			log.Fatalf("Running service: %v", err)
		}
	}()

	log.Infof("ran service")

	return nil
}

func (service *MyService) ControlChannel() <-chan ControlEvent {
	return service.controlChannel
}

func interfaceExists(ctx context.Context, iface string) bool {
	queryService := exec.CommandContext(ctx, "sc", "query", serviceName(iface))
	if err := queryService.Run(); err != nil {
		return false
	} else {
		return true
	}
}

func (configurator *WindowsConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	if interfaceExists(ctx, configurator.helperConfig.Interface) {
		return nil
	}

	installService := exec.CommandContext(ctx, WireGuardBinary, "/installtunnelservice", configurator.helperConfig.WireGuardConfigPath)
	if b, err := installService.CombinedOutput(); err != nil {
		return fmt.Errorf("installing tunnel service: %v: %v", err, string(b))
	} else {
		log.Infof("installed tunnel service, sleeping 3 sec to let it finish")
		time.Sleep(3 * time.Second)
		log.Infof("resuming")
	}

	configurator.wgNeedsRestart = false

	return nil
}

func (configurator *WindowsConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) error {
	return nil
}

func (configurator *WindowsConfigurator) SyncConf(ctx context.Context, cfg *pb.Configuration) error {
	newWireGuardConfig, err := ioutil.ReadFile(configurator.helperConfig.WireGuardConfigPath)
	if err != nil {
		return fmt.Errorf("reading WireGuard config file: %w", err)
	}

	defer func() {
		configurator.oldWireGuardConfig = newWireGuardConfig
		configurator.wgNeedsRestart = true
	}()

	if !configurator.wgNeedsRestart {
		return nil
	}

	if fileActuallyChanged(configurator.oldWireGuardConfig, newWireGuardConfig) {
		log.Debugf("old: %s", string(configurator.oldWireGuardConfig))
		log.Debugf("new: %s", string(newWireGuardConfig))

		commands := [][]string{
			{"net", "stop", serviceName(configurator.helperConfig.Interface)},
			{"net", "start", serviceName(configurator.helperConfig.Interface)},
		}

		return runCommands(ctx, commands)
	}

	return nil
}

func (configurator *WindowsConfigurator) TeardownInterface(ctx context.Context) error {
	if !interfaceExists(ctx, configurator.helperConfig.Interface) {
		log.Info("no interface")
		return nil
	}

	uninstallService := exec.CommandContext(ctx, WireGuardBinary, "/uninstalltunnelservice", configurator.helperConfig.Interface)

	b, err := uninstallService.CombinedOutput()
	if err != nil {
		return fmt.Errorf("uninstalling tunnel service: %v: %v", err, string(b))
	} else {
		log.Infof("uninstalled tunnel service (sleeping 3 sec to let it finish)")
		time.Sleep(3 * time.Second)
		log.Infof("resuming")
	}

	return nil
}

func serviceName(interfaceName string) string {
	return fmt.Sprintf("WireGuardTunnel$%s", interfaceName)
}

func fileActuallyChanged(old, new []byte) bool {
	if old == nil || new == nil {
		return true
	}

	return !bytes.Equal(old, new)
}

func NewService() *MyService {
	return &MyService{controlChannel: make(chan ControlEvent, 100)}
}

func (service *MyService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	log.Infof("service started with args: %v", args)
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				service.controlChannel <- Stop
				break loop
			default:
				log.Errorf("unexpected control request #%d", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}
