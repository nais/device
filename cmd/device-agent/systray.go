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
	"time"

	"github.com/nais/device/device-agent/auth"
	"github.com/nais/device/device-agent/filesystem"
	"github.com/nais/device/device-agent/open"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
)

type ProgramState int

const (
	StateDisconnected ProgramState = iota
	StateNewVersion
	StateBootstrapping
	StateConnected
	StateDisconnecting
	StateUnhealthy
	StateQuitting
	StateSavingConfiguration
	StateAuthenticating
	StateSyncConfig
	StateHealthCheck
)

const (
	versionCheckInterval      = 2 * time.Minute
	syncConfigInterval        = 30 * time.Second
	initialGatewayRefreshWait = 2 * time.Second
	initialConnectWait        = initialGatewayRefreshWait
	healthCheckInterval       = 10 * time.Second
)

func (state ProgramState) String() string {
	switch state {
	case StateDisconnected:
		return "Disconnected"
	case StateBootstrapping:
		return "Bootstrapping..."
	case StateAuthenticating:
		return "Authenticating..."
	case StateSyncConfig:
		fallthrough
	case StateHealthCheck:
		fallthrough
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
		return "Unknown state >_<"
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

	rc, err := runtimeconfig.New(cfg, context.Background())
	if err != nil {
		notify("Unable to start naisdevice, check logs for details")
		return
	}

	go gui.EventLoop()
	go checkVersion(versionCheckInterval, gui)
	mainloop(gui, rc)
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
			stateChange <- StateBootstrapping
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

func mainloop(gui *Gui, rc *runtimeconfig.RuntimeConfig) {
	var err error

	syncConfigTicker := time.NewTicker(syncConfigInterval)
	healthCheckTicker := time.NewTicker(healthCheckInterval)

	state := StateDisconnected
	stateChange := make(chan ProgramState, 64)
	gui.ProgramState <- state

	for {
		select {
		case guiEvent := <-gui.Events:
			handleGuiEvent(guiEvent, state, stateChange)

		case <-syncConfigTicker.C:
			if state == StateConnected {
				stateChange <- StateSyncConfig
			}

		case <-healthCheckTicker.C:
			if state == StateConnected {
				stateChange <- StateHealthCheck
			}

		case state = <-stateChange:
			gui.ProgramState <- state

			switch state {
			case StateBootstrapping:
				if rc.BootstrapConfig != nil {
					log.Infof("Already bootstrapped")
				} else {
					ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
					rc.BootstrapConfig, err = runtimeconfig.EnsureBootstrapping(rc, ctx)
					cancel()
					if err != nil {
						notify(fmt.Sprintf("Error during bootstrap: %v", err))
						stateChange <- StateDisconnected
						continue
					}
				}

				err = WriteConfigFile(rc.Config.WireGuardConfigPath, *rc)
				if err != nil {
					err = fmt.Errorf("unable to write WireGuard configuration file: %w", err)
					notify(err.Error())
					stateChange <- StateDisconnecting
					continue
				}

				time.Sleep(initialConnectWait) // allow wireguard to syncconf
				stateChange <- StateAuthenticating

			case StateAuthenticating:
				if rc.SessionInfo != nil && !rc.SessionInfo.Expired() {
					log.Infof("Already have a valid session")
				} else {
					log.Infof("No valid session, authenticating")
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
					if rc.SessionInfo, err = auth.EnsureAuth(rc.SessionInfo, ctx, rc.Config.APIServer, rc.Config.Platform, rc.Serial); err != nil {
						log.Errorf("Authenticating with apiserver: %v", err)
						stateChange <- StateDisconnecting
						cancel()
						continue
					}
					cancel()
				}
				connectedTime = time.Now()
				stateChange <- StateSyncConfig
				continue

			case StateConnected:
			case StateDisconnected:
			case StateQuitting:
				fallthrough

			case StateDisconnecting:
				if rc != nil {
					rc.Gateways = make(apiserver.Gateways, 0)
					gui.Gateways <- rc.GetGateways()
					err := DeleteConfigFile(rc.Config.WireGuardConfigPath)
					if err != nil {
						notify("error synchronizing WireGuard config: %s", err)
					}
				}
				stateChange <- StateDisconnected

				if state == StateQuitting {
					systray.Quit()
				}
			case StateHealthCheck:
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
				stateChange <- StateSavingConfiguration

			case StateSyncConfig:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				gateways, err := apiserver.GetDeviceConfig(rc.SessionInfo.Key, rc.Config.APIServer, ctx)
				cancel()

				if errors.Is(err, &apiserver.UnauthorizedError{}) {
					log.Errorf("Getting device config: %v", err)
					log.Errorf("Assuming invalid session; disconnecting.")
					rc.SessionInfo = nil
					stateChange <- StateDisconnecting
					continue
				}

				if errors.Is(err, &apiserver.UnhealthyError{}) {
					gui.ProgramState <- StateUnhealthy
					log.Errorf("Device is not healthy: %v", err)
					stateChange <- StateDisconnecting
					continue
				}

				if err != nil {
					log.Errorf("Unable to get gateway config: %v", err)
					continue
				}

				rc.UpdateGateways(gateways)
				gui.Gateways <- rc.GetGateways()
				stateChange <- StateSavingConfiguration

			case StateSavingConfiguration:
				// TODO: Bør vi egentlig skrive fila på nytt hvert 10 sekund om det ikke er endringer?
				err = WriteConfigFile(rc.Config.WireGuardConfigPath, *rc)
				if err != nil {
					err = fmt.Errorf("unable to write WireGuard configuration file: %w", err)
					notify(err.Error())
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

func onExit() {
	// This is where we clean up
}

func ping(addr string) error {
	c, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", addr, "3000"), 2*time.Second)
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
