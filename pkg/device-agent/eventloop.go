package device_agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/auth"
	"github.com/nais/device/device-agent/runtimeconfig"
	pb "github.com/nais/device/pkg/protobuf"
	log "github.com/sirupsen/logrus"
)

var (
	connectedTime         = time.Now()
	lastConfigurationFile string
)

const (
	versionCheckInterval      = 1 * time.Hour
	syncConfigInterval        = 5 * time.Minute
	initialGatewayRefreshWait = 2 * time.Second
	initialConnectWait        = initialGatewayRefreshWait
	healthCheckInterval       = 20 * time.Second
)

func (das *DeviceAgentServer) EventLoop(rc *runtimeconfig.RuntimeConfig) {
	var err error

	syncConfigTicker := time.NewTicker(syncConfigInterval)
	healthCheckTicker := time.NewTicker(healthCheckInterval)

	status := &pb.AgentStatus{}
	das.stateChange <- status.ConnectionState

	for {
		das.UpdateAgentStatus(status)

		select {
		case <-syncConfigTicker.C:
			if status.ConnectionState == pb.AgentState_Connected {
				das.stateChange <- pb.AgentState_SyncConfig
			}

		case <-healthCheckTicker.C:
			if status.ConnectionState == pb.AgentState_Connected {
				das.stateChange <- pb.AgentState_HealthCheck
			}

		case status.ConnectionState = <-das.stateChange:
			log.Infof("state changed to %s", status.ConnectionState)

			switch status.ConnectionState {
			case pb.AgentState_Bootstrapping:
				if rc.BootstrapConfig != nil {
					log.Infof("Already bootstrapped")
				} else {
					ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
					rc.BootstrapConfig, err = runtimeconfig.EnsureBootstrapping(rc, ctx)
					cancel()
					if err != nil {
						notify(fmt.Sprintf("Error during bootstrap: %v", err))
						das.stateChange <- pb.AgentState_Disconnecting
						continue
					}
				}

				err := saveConfig(*rc)
				if err != nil {
					log.Error(err.Error())
					notify(err.Error())
					das.stateChange <- pb.AgentState_Disconnecting
					continue
				}

				time.Sleep(initialConnectWait) // allow wireguard to syncconf
				das.stateChange <- pb.AgentState_Authenticating

			case pb.AgentState_Authenticating:
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
						das.stateChange <- pb.AgentState_Disconnecting
						continue
					}
				}
				connectedTime = time.Now()
				das.stateChange <- pb.AgentState_SyncConfig
				continue

			case pb.AgentState_Connected:
			case pb.AgentState_Disconnected:
				log.Info("making sure no previous WireGuard config exists")
				_ = DeleteConfigFile(rc.Config.WireGuardConfigPath)
				status.Gateways = make([]*pb.Gateway, 0)

			case pb.AgentState_Quitting:
				_ = DeleteConfigFile(rc.Config.WireGuardConfigPath)
				//noinspection GoDeferInLoop
				defer systray.Quit()
				return

			case pb.AgentState_Disconnecting:
				err := DeleteConfigFile(rc.Config.WireGuardConfigPath)
				if err != nil {
					notify("error synchronizing WireGuard config: %s", err)
				}
				das.stateChange <- pb.AgentState_Disconnected

			case pb.AgentState_HealthCheck:
				for _, gw := range rc.GetGateways() {
					err := ping(gw.IP)
					if err == nil {
						gw.Healthy = true
						log.Debugf("Successfully pinged gateway %v with ip: %v", gw.Name, gw.IP)
					} else {
						gw.Healthy = false
						log.Infof("unable to ping host %s: %v", gw.IP, err)
					}
				}
				das.stateChange <- pb.AgentState_Connected
				status.Gateways = ApiGatewaysToProtobufGateways(rc.Gateways)
				das.UpdateAgentStatus(status)
				// trigger configuration save here if health checks are supposed to alter routes

			case pb.AgentState_SyncConfig:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				gateways, err := apiserver.GetDeviceConfig(rc.SessionInfo.Key, rc.Config.APIServer, ctx)
				cancel()

				switch {
				case errors.Is(err, &apiserver.UnauthorizedError{}):
					log.Errorf("Unauthorized access from apiserver: %v", err)
					log.Errorf("Assuming invalid session; disconnecting.")
					rc.SessionInfo = nil
					das.stateChange <- pb.AgentState_Disconnecting
					continue

				case errors.Is(err, &apiserver.UnhealthyError{}):
					// TODO produce unhealthy status message to "even watcher" stream

					log.Errorf("Device is not healthy: %v", err)
					// TODO consider moving all notify calls to systray code
					notify("No access as your device is unhealthy. Run '/msg @Kolide status' on Slack and fix the errors")
					das.stateChange <- pb.AgentState_Unhealthy
					continue

				case err != nil:
					log.Errorf("Unable to get gateway config: %v", err)
					das.stateChange <- pb.AgentState_HealthCheck
					continue
				}

				rc.UpdateGateways(gateways)
				status.Gateways = ApiGatewaysToProtobufGateways(rc.Gateways)
				das.UpdateAgentStatus(status)

				err = saveConfig(*rc)
				if err != nil {
					log.Error(err.Error())
					notify(err.Error())
					das.stateChange <- pb.AgentState_Disconnecting
				} else {
					das.stateChange <- pb.AgentState_HealthCheck
				}

			case pb.AgentState_Unhealthy:
			}
		}
	}
}

func ApiGatewaysToProtobufGateways(apigws apiserver.Gateways) []*pb.Gateway {
	var pbgws []*pb.Gateway

	for _, apigw := range apigws {
		pbgws = append(pbgws, &pb.Gateway{
			Name:                     apigw.Name,
			Healthy:                  apigw.Healthy,
			PublicKey:                apigw.PublicKey,
			Endpoint:                 apigw.Endpoint,
			Ip:                       apigw.IP,
			Routes:                   apigw.Routes,
			RequiresPrivilegedAccess: apigw.RequiresPrivilegedAccess,
			AccessGroupIDs:           nil,
		})
	}

	return pbgws
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
