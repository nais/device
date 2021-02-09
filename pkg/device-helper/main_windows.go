package device_helper

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nais/device/cmd/device-agent-helper"
	"github.com/nais/device/pkg/bootstrap"
	"github.com/nais/device/pkg/logger"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	WireGuardBinary = `c:\Program Files\WireGuard\wireguard.exe`
	ServiceName     = "naisdevice-agent-helper"
)

type MyService struct {
	controlChannel chan main.ControlEvent
}

func (service *MyService) ControlChannel() <-chan main.ControlEvent {
	return service.controlChannel
}

func PlatformInit(cfg *main.Config) {
	interactive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("Checking if session is interactive: %v", err)
	}

	if interactive {
		return
	}

	s := NewService()
	main.myService = s
	go func() {
		err := svc.Run(ServiceName, s)
		if err != nil {
			log.Fatalf("Running service: %v", err)
		}
	}()
}

func interfaceExists(ctx context.Context, cfg main.Config) bool {
	queryService := exec.CommandContext(ctx, "sc", "query", serviceName(cfg.Interface))
	if err := queryService.Run(); err != nil {
		return false
	} else {
		return true
	}
}

func setupInterface(ctx context.Context, cfg main.Config, bootstrapConfig *bootstrap.Config) error {
	if interfaceExists(ctx, cfg) {
		return nil
	}

	installService := exec.CommandContext(ctx, WireGuardBinary, "/installtunnelservice", cfg.WireGuardConfigPath)
	if b, err := installService.CombinedOutput(); err != nil {
		return fmt.Errorf("installing tunnel service: %v: %v", err, string(b))
	} else {
		log.Infof("installed tunnel service, sleeping 3 sec to let it finish")
		time.Sleep(3 * time.Second)
		log.Infof("resuming")
	}

	return nil
}

var oldWireGuardConfig []byte

func syncConf(cfg main.Config, ctx context.Context) error {
	newWireGuardConfig, err := ioutil.ReadFile(cfg.WireGuardConfigPath)
	if err != nil {
		return fmt.Errorf("reading WireGuard config file: %w", err)
	}

	if fileActuallyChanged(oldWireGuardConfig, newWireGuardConfig) {
		oldWireGuardConfig = newWireGuardConfig

		commands := [][]string{
			{"net", "stop", serviceName(cfg.Interface)},
			{"net", "start", serviceName(cfg.Interface)},
		}

		return main.runCommands(ctx, commands)
	}

	return nil
}

func teardownInterface(ctx context.Context, cfg main.Config) {
	if !interfaceExists(ctx, cfg) {
		log.Info("no interface")
		return
	}

	uninstallService := exec.CommandContext(ctx, WireGuardBinary, "/uninstalltunnelservice", cfg.Interface)
	if b, err := uninstallService.CombinedOutput(); err != nil {
		log.Warnf("uninstalling tunnel service: %v: %v", err, string(b))
	} else {
		log.Infof("uninstalled tunnel service (sleeping 3 sec to let it finish)")
		time.Sleep(3 * time.Second)
		log.Infof("resuming")
	}
}

func prerequisites() error {
	if err := main.filesExist(WireGuardBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %w", err)
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

func exePath() (string, error) {
	program := os.Args[0]
	absoluteProgramPath, err := filepath.Abs(program)
	if err != nil {
		return "", fmt.Errorf("getting absolute program path: %w", err)
	}

	if filepath.Ext(absoluteProgramPath) == "" {
		absoluteProgramPath += ".exe"
	}

	fi, err := os.Stat(absoluteProgramPath)
	if err != nil {
		return "", fmt.Errorf("getting file stats: %w", err)
	}

	if fi.Mode().IsDir() {
		return "", fmt.Errorf("%v is a directory", absoluteProgramPath)
	}

	return absoluteProgramPath, nil
}

func uninstallService() {
	log.Info("Uninstalling service: %s", ServiceName)
	m, err := mgr.Connect()
	if err != nil {
		log.Error("Connecting to Service Manager: %v", err)
		return
	}
	defer m.Disconnect()

	s, err := m.OpenService(ServiceName)
	if err != nil {
		log.Infof("Opening service \"%v\": %v", ServiceName, err)
		return
	}
	defer s.Close()

	_, err = s.Control(svc.Stop)
	if err != nil {
		log.Warnf("Stopping service: %v", err)
	}

	err = s.Delete()
	if err != nil {
		log.Warnf("Deleting service: %v", err)
	}
}

func NewService() *MyService {
	return &MyService{controlChannel: make(chan main.ControlEvent, 100)}
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
				service.controlChannel <- main.Stop
				break loop
			default:
				log.Errorf("unexpected control request #%d", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}
