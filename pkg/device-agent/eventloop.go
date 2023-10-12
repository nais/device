package device_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/ioconvenience"

	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/device-agent/auth"
	"github.com/nais/device/pkg/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/notify"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/version"
)

const (
	healthCheckInterval    = 20 * time.Second // how often to healthcheck gateways
	syncConfigDialTimeout  = 1 * time.Second  // sleep time between failed configuration syncs
	versionCheckInterval   = 1 * time.Hour    // how often to check for a new version of naisdevice
	versionCheckTimeout    = 3 * time.Second  // timeout for new version check
	getSerialTimeout       = 2 * time.Second  // timeout for getting device serial from helper
	authFlowTimeout        = 1 * time.Minute  // total timeout for authenticating user (AAD login in browser, redirect to localhost, exchange code for token)
	apiServerRetryInterval = time.Second * 5
)

func (das *DeviceAgentServer) ConfigureHelper(ctx context.Context, rc runtimeconfig.RuntimeConfig, gateways []*pb.Gateway) error {
	_, err := das.DeviceHelper.Configure(ctx, das.rc.BuildHelperConfiguration(gateways))
	return err
}

func (das *DeviceAgentServer) syncConfigLoop(ctx context.Context, gateways chan<- []*pb.Gateway, notifyConnected chan struct{}) error {
	dialContext, cancel := context.WithTimeout(ctx, syncConfigDialTimeout)
	defer cancel()

	conn, err := das.rc.DialAPIServer(dialContext)
	if err != nil {
		return grpcstatus.Errorf(codes.Unavailable, err.Error())
	}
	log.Infof("Connected to API server")

	defer conn.Close()

	apiserverClient := pb.NewAPIServerClient(conn)

	session, err := das.rc.GetTenantSession()
	if err != nil {
		return err
	}

	if session.Expired() {
		serial, err := das.getSerial(ctx)
		if err != nil {
			return err
		}

		token, err := das.rc.GetToken(ctx)
		if err != nil {
			return err
		}

		loginResponse, err := apiserverClient.Login(ctx, &pb.APIServerLoginRequest{
			Token:    token,
			Platform: config.Platform,
			Serial:   serial,
			Version:  version.Version,
		})
		if err != nil {
			return err
		}

		if err := das.rc.SetTenantSession(loginResponse.Session); err != nil {
			return err
		}
		session = loginResponse.Session
	}

	streamContext, cancel := context.WithDeadline(ctx, session.Expiry.AsTime())
	defer cancel()

	stream, err := apiserverClient.GetDeviceConfiguration(streamContext, &pb.GetDeviceConfigurationRequest{
		SessionKey: session.Key,
	})
	if err != nil {
		return err
	}

	log.Infof("Gateway configuration stream established")

	// notify calling function that we are connected
	select {
	case notifyConnected <- struct{}{}:
	default:
	}

	for {
		cfg, err := stream.Recv()
		if err != nil {
			return err
		}

		log.Infof("Received gateway configuration from API server")

		switch cfg.Status {
		case pb.DeviceConfigurationStatus_InvalidSession:
			log.Errorf("Unauthorized access from apiserver: %v", err)
			log.Errorf("Assuming invalid session; disconnecting.")
			return fmt.Errorf("config status: %v", cfg.Status)
		case pb.DeviceConfigurationStatus_DeviceUnhealthy:
			log.Errorf("Device is not healthy: %v", err)
			// TODO consider moving all notify calls to systray code
			notify.Errorf("No access as your device is unhealthy. Run '/msg @Kolide status' on Slack and fix the errors")
			das.stateChange <- pb.AgentState_Unhealthy
			continue
		case pb.DeviceConfigurationStatus_DeviceHealthy:
			log.Infof("Device is healthy; server pushed %d gateways", len(cfg.Gateways))
		default:
		}

		gateways <- cfg.Gateways
	}
}

func (das *DeviceAgentServer) getSerial(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, getSerialTimeout)
	defer cancel()
	serial, err := das.DeviceHelper.GetSerial(ctx, &pb.GetSerialRequest{})
	return serial.GetSerial(), err
}

func (das *DeviceAgentServer) EventLoop(programContext context.Context) {
	var err error
	var syncctx context.Context
	var synccancel context.CancelFunc

	gateways := make(chan []*pb.Gateway, 16)
	status := &pb.AgentStatus{}

	healthCheckTicker := time.NewTicker(healthCheckInterval)
	versionCheckTicker := time.NewTicker(5 * time.Second)

	autoConnectTriggered := false

	das.stateChange <- status.ConnectionState

	status.Tenants = das.rc.Tenants()

	for {
		das.UpdateAgentStatus(status)

		if das.rc.Tenants() == nil {
			notify.Errorf("No tenants configured. Please configure tenants in the configuration file.")
			return
		}

		select {
		case <-programContext.Done():
			log.Infof("EventLoop: context done")
			return

		case <-versionCheckTicker.C:
			ctx, cancel := context.WithTimeout(programContext, versionCheckTimeout)
			status.NewVersionAvailable, err = newVersionAvailable(ctx)
			cancel()

			if err != nil {
				log.Errorf("check for new version: %s", err)
				break
			}

			if status.NewVersionAvailable {
				notify.Infof("New version of device agent available: https://doc.nais.io/device/update/")
				versionCheckTicker.Stop()
			} else {
				versionCheckTicker.Reset(versionCheckInterval)
			}

		case <-healthCheckTicker.C:
			healthCheckTicker.Reset(healthCheckInterval)
			if status.ConnectionState != pb.AgentState_Connected {
				break
			}

			wg := &sync.WaitGroup{}

			total := len(status.GetGateways())
			log.Debugf("Pinging %d gateways...", total)
			for i, gw := range status.GetGateways() {
				wg.Add(1)
				go func(i int, gw *pb.Gateway) {
					err := ping(gw.Ipv4)
					pos := fmt.Sprintf("[%02d/%02d]", i+1, total)
					if err == nil {
						gw.Healthy = true
						log.Debugf("%s %s: successfully pinged %v", pos, gw.Name, gw.Ipv4)
					} else {
						gw.Healthy = false
						log.Debugf("%s %s: unable to ping %s: %v", pos, gw.Name, gw.Ipv4, err)
					}
					wg.Done()
				}(i, gw)
			}
			wg.Wait()

		case gws := <-gateways:
			if syncctx == nil {
				log.Errorf("BUG: synchronization context is nil while updating gateways")
				break
			}

			pb.MergeGatewayHealth(gws, status.GetGateways())
			status.Gateways = gws

			ctx, cancel := context.WithTimeout(syncctx, helperTimeout)
			err = das.ConfigureHelper(ctx, das.rc, append(
				[]*pb.Gateway{
					das.rc.APIServerPeer(),
				},
				status.GetGateways()...,
			))
			cancel()

			if err != nil {
				notify.Errorf(err.Error())
				das.stateChange <- pb.AgentState_Disconnecting
			} else {
				das.stateChange <- pb.AgentState_Connected
			}

		case newState := <-das.stateChange:
			previousState := status.ConnectionState
			status.ConnectionState = newState
			log.Infof("state changed from %s to %s", previousState, status.ConnectionState)

			switch status.ConnectionState {
			case pb.AgentState_Bootstrapping:
				if previousState != pb.AgentState_Authenticating {
					log.Errorf("probably concurrency issue: came here from invalid state %q, aborting", previousState)
					das.stateChange <- pb.AgentState_Disconnecting
					break
				}

				if err := das.rc.LoadEnrollConfig(); err == nil {
					log.Infof("Loaded enroll")
				} else {
					log.Infof("Unable to load enroll config: %s", err)
					log.Infof("Enrolling device")
					ctx, cancel := context.WithTimeout(programContext, 1*time.Minute)
					serial, err := das.getSerial(ctx)
					if err != nil {
						notify.Errorf("Unable to get serial number: %v", err)
						das.stateChange <- pb.AgentState_Disconnecting
						cancel()
						continue
					}

					err = das.rc.EnsureEnrolled(ctx, serial)

					cancel()
					if err != nil {
						notify.Errorf("Bootstrap: %v", err)
						das.stateChange <- pb.AgentState_Disconnecting
						continue
					}
				}

				ctx, cancel := context.WithTimeout(programContext, helperTimeout)
				err = das.ConfigureHelper(ctx, das.rc, []*pb.Gateway{
					das.rc.APIServerPeer(),
				})
				cancel()

				if err != nil {
					notify.Errorf(err.Error())
					das.stateChange <- pb.AgentState_Disconnecting
					continue
				}

				status.ConnectedSince = timestamppb.Now()

				notifyConnected := make(chan struct{})
				syncctx, synccancel = context.WithCancel(programContext)
				go func() {
					attempt := 0
					for syncctx.Err() == nil {
						attempt++
						log.Infof("[attempt %d] Setting up gateway configuration stream...", attempt)
						err := das.syncConfigLoop(syncctx, gateways, notifyConnected)

						switch grpcstatus.Code(err) {
						case codes.OK:
							attempt = 0
						case codes.Unavailable:
							log.Warnf("Synchronize config: not connected to API server: %v", err)
							time.Sleep(apiServerRetryInterval)
						case codes.Unauthenticated:
							log.Errorf("Logging in: %s", err)
							das.rc.SetToken(nil)
							log.Error("Cleaned up old tokens")
							fallthrough
						default:
							notify.Errorf(err.Error())
							if das.AgentStatus.ConnectionState != pb.AgentState_Disconnecting {
								das.stateChange <- pb.AgentState_Disconnecting
							}
							synccancel()
						}
					}

					log.Infof("Gateway config synchronization loop: %s", syncctx.Err())
					syncctx = nil
					synccancel()
				}()

				<-notifyConnected
				das.stateChange <- pb.AgentState_Connected
				continue

			case pb.AgentState_Authenticating:
				if previousState != pb.AgentState_Disconnected {
					log.Errorf("probably concurrency issue: came here from invalid state %q, aborting", previousState)
					das.stateChange <- pb.AgentState_Disconnecting
					break
				}

				session, _ := das.rc.GetTenantSession()
				if !session.Expired() {
					das.stateChange <- pb.AgentState_Bootstrapping
					break
				}

				ctx, cancel := context.WithTimeout(programContext, authFlowTimeout)
				oauth2Config := das.Config.OAuth2Config(das.rc.GetActiveTenant().AuthProvider)
				token, err := auth.GetDeviceAgentToken(ctx, oauth2Config, das.Config.GoogleAuthServerAddress)
				cancel()
				if err != nil {
					notify.Errorf("Get token: %v", err)
					das.stateChange <- pb.AgentState_Disconnected
					break
				}

				das.rc.SetToken(token)

				das.stateChange <- pb.AgentState_Bootstrapping

			case pb.AgentState_Connected:
				healthCheckTicker.Reset(1 * time.Second)

			case pb.AgentState_Disconnected:
				status.Gateways = make([]*pb.Gateway, 0)
				if das.Config.AgentConfiguration.AutoConnect && !autoConnectTriggered {
					autoConnectTriggered = true
					das.stateChange <- pb.AgentState_Authenticating
				}

				das.rc.SetToken(nil)
				das.rc.ResetEnrollConfig()

			case pb.AgentState_Quitting:
				return

			case pb.AgentState_Disconnecting:
				if synccancel != nil {
					synccancel() // cancel streaming gateway updates
				}
				log.Info("Tearing down network connections through device-helper...")
				ctx, cancel := context.WithTimeout(programContext, helperTimeout)
				_, err = das.DeviceHelper.Teardown(ctx, &pb.TeardownRequest{})
				cancel()

				if err != nil {
					notify.Errorf(err.Error())
				}

				das.stateChange <- pb.AgentState_Disconnected

			case pb.AgentState_Unhealthy:

			case pb.AgentState_AgentConfigurationChanged:
				das.stateChange <- previousState
			}
		}
	}
}

func newVersionAvailable(ctx context.Context) (bool, error) {
	type response struct {
		Tag string `json:"tag_name"`
	}

	log.Info("Checking release version on github")

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/nais/device/releases/latest", nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("retrieve current release version: %s", err)
	}

	defer ioconvenience.CloseWithLog(resp.Body)

	res := &response{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(res)
	if err != nil {
		return false, fmt.Errorf("unmarshal response: %s", err)
	}

	if version.Version != res.Tag {
		return true, nil
	}

	return false, nil
}

func ping(addr string) error {
	c, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", addr, "3000"), 2*time.Second)
	if err != nil {
		return err
	}

	err = c.Close()
	if err != nil {
		log.Errorf("closing ping connection: %v", err)
	}

	return nil
}
