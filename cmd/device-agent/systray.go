package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nais/device/device-agent/filesystem"
	"github.com/nais/device/device-agent/open"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/auth"
	"github.com/nais/device/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
)

type ProgramState int

const (
	StateDisconnected ProgramState = iota
	StateNewVersion
	StateBootstrapping
	StateConnecting
	StateConnected
	StateDisconnecting
	StateUnhealthy
	StateQuitting
	StateSavingConfiguration
	StateAuthenticating
	StateSyncConfig
	StateHealthCheck
	StateRunning
)

const (
	versionCheckInterval      = 2 * time.Minute
	gatewayRefreshInterval    = 10 * time.Second
	initialGatewayRefreshWait = 2 * time.Second
	initialConnectWait        = initialGatewayRefreshWait
	healthCheckInterval       = 5 * time.Second
)

func (state ProgramState) String() string {
	switch state {
	case StateDisconnected:
		return "Disconnected"
	case StateBootstrapping:
		return "Bootstrapping..."
	case StateConnecting:
		return "Connecting..."
	case StateSavingConfiguration:
		fallthrough
	case StateConnected:
		return fmt.Sprintf("Connected since %s", connectedTime.Format(time.Kitchen))
	case StateUnhealthy:
		return "Device is unhealthy..."
	case StateDisconnecting:
		return "Disconnecting..."
	case StateQuitting:
		return "Quitting..."
	default:
		return "Unknown state!!!"
	}
}

var (
	connectedTime = time.Now()
)

func onReady() {
	gui := NewGUI()

	if err := filesystem.EnsurePrerequisites(&cfg); err != nil {
		notify(fmt.Sprintf("Missing prerequisites: %s", err))
	}

	go gui.EventLoop()
	go checkVersion(versionCheckInterval, gui)
	mainloop(gui)
}

func handleGuiEvent(guiEvent GuiEvent, state ProgramState, stateChange chan ProgramState) {
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
		if state == StateDisconnected {
			stateChange <- StateConnecting
		} else {
			stateChange <- StateDisconnecting
		}
	case HelperLogClicked:
		err := open.Open("/Library/Logs/device-agent-helper-err.log")
		if err != nil {
			log.Warn("opening device agent helper log: %w", err)
		}
	case DeviceLogClicked:
		homedir, err := os.UserHomeDir()
		if err != nil {
			log.Warn("finding user's home directory", err)
		}
		err = open.Open(filepath.Join(homedir, "Library", "Logs", "device-agent.log"))
		if err != nil {
			log.Warn("opening device agent log: %w", err)
		}

	case QuitClicked:
		stateChange <- StateQuitting
	}

}

func mainloop(gui *Gui) {
	var rc *runtimeconfig.RuntimeConfig
	var err error

	syncConfigTicker := time.NewTicker(gatewayRefreshInterval)
	healthCheckTicker := time.NewTicker(healthCheckInterval)

	once := sync.Once{}
	stop := make(chan interface{}, 1)
	state := StateDisconnected
	stateChange := make(chan ProgramState, 64)
	gui.ProgramState <- state

	for {
		select {
		case guiEvent := <-gui.Events:
			handleGuiEvent(guiEvent, state, stateChange)

		case <-syncConfigTicker.C:
			if state == StateRunning {
				stateChange <- StateSyncConfig
			}

		case <-healthCheckTicker.C:
			if state == StateRunning {
				stateChange <- StateHealthCheck
			}

		case state = <-stateChange:
			gui.ProgramState <- state

			switch state {
			case StateDisconnected:
			case StateBootstrapping:
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute*3)
				rc, err = runtimeconfig.New(cfg, ctx)
				cancel()
				if err != nil {
					notify(err.Error())
					stateChange <- StateDisconnected
					continue
				}
				stateChange <- StateConnecting

			case StateConnecting:
				if rc == nil {
					stateChange <- StateBootstrapping
					continue
				}
				err = WriteConfigFile(rc.Config.WireGuardConfigPath, *rc)
				if err != nil {
					err = fmt.Errorf("unable to write WireGuard configuration file: %w", err)
					notify(err.Error())
					stateChange <- StateDisconnected
					continue
				}
				time.Sleep(initialConnectWait) // allow wireguard to syncconf
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
				rc.SessionInfo, err = auth.EnsureAuth(rc.SessionInfo, ctx, rc.Config.APIServer, rc.Config.Platform, rc.Serial)
				cancel()

				if err == nil {
					go synchronizeGateways(gatewayRefreshInterval, stop, rc, stateChange)
					stateChange <- StateConnected
					notify("connected")
					connectedTime = time.Now()
					once.Do(func() {
						go checkGatewayHealth(healthCheckInterval, rc, gui)
					})
				} else {
					stateChange <- StateDisconnected
					notify(err.Error())
				}

			case StateConnected:
			case StateQuitting:
				fallthrough

			case StateDisconnecting:
				stop <- new(interface{})

				if rc != nil {
					rc.Gateways = make(apiserver.Gateways, 0)
					err := DeleteConfigFile(rc.Config.WireGuardConfigPath)
					if err != nil {
						notify("error synchronizing WireGuard config: %s", err)
					}
				}
				stateChange <- StateDisconnected

				if state == StateQuitting {
					systray.Quit()
				}

			case StateSavingConfiguration:
				// TODO: Bør vi egentlig skrive fila på nytt hvert 10 sekund om det ikke er endringer?
				err = WriteConfigFile(rc.Config.WireGuardConfigPath, *rc)
				if err != nil {
					err = fmt.Errorf("unable to write WireGuard configuration file: %w", err)
					notify(err.Error())
					return
				}
				stateChange <- StateConnected

			case StateUnhealthy:
			}
		}
	}
}

func checkGatewayHealth(interval time.Duration, rc *runtimeconfig.RuntimeConfig, gui *Gui) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			for _, gw := range rc.GetGateways() {
				err := ping(gw.IP)
				if err == nil {
					gw.Healthy = true
					log.Debugf("Successfully pinged gateway %v with ip: %v", gw.Name, gw.IP)
				} else {
					gw.Healthy = false
					log.Errorf("unable to ping host %s: %v", gw.IP, err)
				}
			}
			gui.Gateways <- rc.GetGateways()
		}
	}
}

func checkVersion(interval time.Duration, gui *Gui) {
	type response struct {
		Tag string `json:"tag_name"`
	}

	ticker := time.NewTicker(1 * time.Nanosecond)
	for range ticker.C {
		ticker.Stop()
		ticker = time.NewTicker(interval)
		log.Info("Checking release version on github")
		resp, err := http.Get("https://api.github.com/repos/nais/device/releases/latest")
		if err != nil {
			log.Errorf("Unable to retrieve current release version %s", err)
			continue
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		res := response{}
		err = json.Unmarshal(body, &res)
		if err != nil {
			log.Errorf("unable to unmarshall response: %s", err)
			continue
		}
		if version.Version != res.Tag {
			gui.NewVersionAvailable <- true
			notify("New version of device agent available: https://doc.nais.io/device/install#installation")
			return
		}
	}
}

func synchronizeGateways(interval time.Duration, stop chan interface{}, rc *runtimeconfig.RuntimeConfig, stateChange chan ProgramState) {
	// Sleeping whilst waiting for API-server connection; waiting for wireguard to sync configuration
	time.Sleep(initialGatewayRefreshWait)

	ctx, cancel := context.WithTimeout(context.Background(), gatewayRefreshInterval)
	fetchDeviceConfig(ctx, rc, stateChange)
	cancel()

	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), gatewayRefreshInterval)
			fetchDeviceConfig(ctx, rc, stateChange)
			cancel()
		case <-stop:
			return
		}
	}
}

func fetchDeviceConfig(ctx context.Context, rc *runtimeconfig.RuntimeConfig, stateChange chan ProgramState) {
	gateways, err := apiserver.GetDeviceConfig(rc.SessionInfo.Key, rc.Config.APIServer, ctx)

	if ue, ok := err.(*apiserver.UnauthorizedError); ok {
		stateChange <- StateDisconnecting
		log.Errorf("unauthorized access from apiserver: %v", ue)
		log.Errorf("assuming invalid session; disconnecting.")
		return
	}

	if errors.Is(err, &apiserver.UnhealthyError{}) {
		stateChange <- StateUnhealthy
		log.Errorf("fetching device config: %v", err)
		return
	}

	if err != nil {
		log.Errorf("unable to get gateway config: %v", err)
		return
	}

	rc.Gateways = gateways

	stateChange <- StateSavingConfiguration
}

func onExit() {
	// This is where we clean up
}

func ping(addr string) error {
	c, err := net.Dial("tcp", fmt.Sprintf("%s:%s", addr, "3000"))
	if err != nil {
		return err
	}
	c.Close()

	return nil
}

func notify(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	err := beeep.Notify("NAIS device", message, "../Resources/nais-logo-red.png")
	log.Infof("sending message to notification centre: %s", message)
	if err != nil {
		log.Errorf("failed sending message due to error: %s", err)
	}
}
