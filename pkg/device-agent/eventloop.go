package device_agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/auth"
	"github.com/nais/device/device-agent/runtimeconfig"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net"
	"os"
	"time"
)


var (
	connectedTime		 = time.Now()
	lastConfigurationFile string
)

func EventLoop(rc *runtimeconfig.RuntimeConfig, stateChange chan ProgramState) {
	var err error

	syncConfigTicker := time.NewTicker(syncConfigInterval)
	healthCheckTicker := time.NewTicker(healthCheckInterval)

	state := StateDisconnected
	stateChange <- state

	for {
		select {
		case <-syncConfigTicker.C:
			if state == StateConnected {
				stateChange <- StateSyncConfig
			}

		case <-healthCheckTicker.C:
			if state == StateConnected {
				stateChange <- StateHealthCheck
			}

		case state = <-stateChange:
			log.Infof("state changed to %s", state)

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
						stateChange <- StateDisconnecting
						continue
					}
				}

				err := saveConfig(*rc)
				if err != nil {
					log.Error(err.Error())
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
					rc.SessionInfo, err = auth.EnsureAuth(rc.SessionInfo, ctx, rc.Config.APIServer, rc.Config.Platform, rc.Serial)
					cancel()
					if err != nil {
						notify(fmt.Sprintf("Error during authentication: %v", err))

						log.Errorf("Authenticating with apiserver: %v", err)
						stateChange <- StateDisconnecting
						continue
					}
				}
				connectedTime = time.Now()
				stateChange <- StateSyncConfig
				continue

			case StateConnected:
			case StateDisconnected:
				log.Info("making sure no previous WireGuard config exists")
				_ = DeleteConfigFile(rc.Config.WireGuardConfigPath)

			case StateQuitting:
				_ = DeleteConfigFile(rc.Config.WireGuardConfigPath)
				//noinspection GoDeferInLoop
				defer systray.Quit()
				return

			case StateDisconnecting:
				err := DeleteConfigFile(rc.Config.WireGuardConfigPath)
				if err != nil {
					notify("error synchronizing WireGuard config: %s", err)
				}
				stateChange <- StateDisconnected

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
				stateChange <- StateConnected
				// trigger configuration save here if health checks are supposed to alter routes

			case StateSyncConfig:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				gateways, err := apiserver.GetDeviceConfig(rc.SessionInfo.Key, rc.Config.APIServer, ctx)
				cancel()

				switch {
				case errors.Is(err, &apiserver.UnauthorizedError{}):
					log.Errorf("Unauthorized access from apiserver: %v", err)
					log.Errorf("Assuming invalid session; disconnecting.")
					rc.SessionInfo = nil
					stateChange <- StateDisconnecting
					continue

				case errors.Is(err, &apiserver.UnhealthyError{}):
					// TODO produce unhealthy status message to "even watcher" stream

					log.Errorf("Device is not healthy: %v", err)
					// TODO consider moving all notify calls to systray code
					notify("No access as your device is unhealthy. Run '/msg @Kolide status' on Slack and fix the errors")
					stateChange <- StateUnhealthy
					continue

				case err != nil:
					log.Errorf("Unable to get gateway config: %v", err)
					stateChange <- StateHealthCheck
					continue
				}

				rc.UpdateGateways(gateways)

				err = saveConfig(*rc)
				if err != nil {
					log.Error(err.Error())
					notify(err.Error())
					stateChange <- StateDisconnecting
				} else {
					stateChange <- StateHealthCheck
				}

			case StateUnhealthy:
			}
		}
	}
}

func saveConfig(rc runtimeconfig.RuntimeConfig) error {
	cfgbuf := new(bytes.Buffer)
	_, err := rc.Write(cfgbuf)
	if err != nil {
		return fmt.Errorf("unable to create WireGuard configuration: %w", err)
	}
	newConfigurationFile := cfgbuf.String()
	if newConfigurationFile == lastConfigurationFile {
		log.Debugf("skip writing identical configuration file")
		return nil
	}
	err = WriteConfigFile(rc.Config.WireGuardConfigPath, cfgbuf)
	if err != nil {
		return fmt.Errorf("unable to create WireGuard configuration file %s: %w", rc.Config.WireGuardConfigPath, err)
	}
	lastConfigurationFile = newConfigurationFile
	return nil
}
/*
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
			// gui.NewVersionAvailable <- true
			notify("New version of device agent available: https://doc.nais.io/device/install#installation")
			return
		}
	}
}
*/

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

func DeleteConfigFile(path string) error {
	err := os.Remove(path)
	if err != nil && err != os.ErrNotExist {
		return err
	}
	log.Debugf("Removed WireGuard configuration file at %s", path)
	lastConfigurationFile = ""
	return nil
}

func WriteConfigFile(path string, r io.Reader) error {
	cfg, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, cfg, 0600)
	if err != nil {
		return fmt.Errorf("writing WireGuard config to disk: %w", err)
	}

	log.Debugf("Wrote WireGuard config to disk")
	return nil
}
