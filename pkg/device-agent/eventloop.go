package device_agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	clientcert "github.com/nais/device/pkg/client-cert"
	"github.com/nais/device/pkg/device-agent/config"

	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/device-agent/auth"
	"github.com/nais/device/pkg/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/notify"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/version"
)

const (
	healthCheckInterval   = 20 * time.Second   // how often to healthcheck gateways
	syncConfigDialTimeout = 1 * time.Second    // sleep time between failed configuration syncs
	versionCheckInterval  = 1 * time.Hour      // how often to check for a new version of naisdevice
	versionCheckTimeout   = 3 * time.Second    // timeout for new version check
	authFlowTimeout       = 30 * time.Second   // total timeout for authenticating user (AAD login in browser, redirect to localhost, exchange code for token)
	approximateInfinity   = time.Hour * 69_420 // Name describes purpose. Used for renewing microsoft client certs automatically
	certRenewalInterval   = time.Hour * 23     // Microsoft Client certificate validity/renewal interval
	certRenewalBackoff    = time.Second * 10   // Self-explanatory (Microsoft Client certificate)
)

var (
	ExpiredToken      = errors.New("azure ad token expired")
	TokenDoesNotExist = errors.New("azure ad token does not exist")
)

func (das *DeviceAgentServer) ConfigureHelper(ctx context.Context, rc *runtimeconfig.RuntimeConfig, gateways []*pb.Gateway) error {
	_, err := das.DeviceHelper.Configure(ctx, &pb.Configuration{
		PrivateKey: base64.StdEncoding.EncodeToString(rc.PrivateKey),
		DeviceIP:   rc.BootstrapConfig.DeviceIP,
		Gateways:   gateways,
	})
	return err
}

func validateToken(token *oauth2.Token) error {
	if token == nil {
		return TokenDoesNotExist
	} else if time.Now().After(token.Expiry) {
		return ExpiredToken
	}

	return nil
}

func (das *DeviceAgentServer) syncConfigLoop(ctx context.Context, gateways chan<- []*pb.Gateway) error {
	dialContext, cancel := context.WithTimeout(ctx, syncConfigDialTimeout)
	defer cancel()

	log.Infof("Attempting gRPC connection to API server on %s...", das.Config.APIServerGRPCAddress)
	apiserver, err := grpc.DialContext(
		dialContext,
		das.Config.APIServerGRPCAddress,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithReturnConnectionError(),
	)
	log.Infof("Connected to API server")

	if err != nil {
		return fmt.Errorf("connect to API server: %v", err)
	}

	defer apiserver.Close()

	apiserverClient := pb.NewAPIServerClient(apiserver)

	if das.rc.SessionInfo.Expired() {
		loginResponse, err := apiserverClient.Login(ctx, &pb.APIServerLoginRequest{
			Token:    das.rc.Token.AccessToken,
			Platform: config.Platform,
			Serial:   das.rc.Serial,
		})

		if err != nil {
			return err
		}

		das.rc.SessionInfo = loginResponse.Session
	}

	stream, err := apiserverClient.GetDeviceConfiguration(ctx, &pb.GetDeviceConfigurationRequest{
		SessionKey: das.rc.SessionInfo.Key,
	})

	if err != nil {
		return err
	}

	log.Infof("Gateway configuration stream established")

	for {
		config, err := stream.Recv()
		if err != nil {
			return err
		}

		log.Infof("Received gateway configuration from API server")

		switch config.Status {
		case pb.DeviceConfigurationStatus_InvalidSession:
			log.Errorf("Unauthorized access from apiserver: %v", err)
			log.Errorf("Assuming invalid session; disconnecting.")
			das.stateChange <- pb.AgentState_Disconnecting
			continue
		case pb.DeviceConfigurationStatus_DeviceUnhealthy:
			log.Errorf("Device is not healthy: %v", err)
			// TODO consider moving all notify calls to systray code
			notify.Errorf("No access as your device is unhealthy. Run '/msg @Kolide status' on Slack and fix the errors")
			das.stateChange <- pb.AgentState_Unhealthy
			continue
		case pb.DeviceConfigurationStatus_DeviceHealthy:
			log.Infof("Device is healthy; server pushed %d gateways", len(config.Gateways))
		default:
		}

		gateways <- config.Gateways
	}
}

func (das *DeviceAgentServer) EventLoop() {
	var err error
	var syncctx context.Context
	var synccancel context.CancelFunc

	gateways := make(chan []*pb.Gateway, 16)
	status := &pb.AgentStatus{}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	healthCheckTicker := time.NewTicker(healthCheckInterval)
	versionCheckTicker := time.NewTicker(5 * time.Second)
	certRenewalTicker := time.NewTicker(approximateInfinity)

	autoConnectTriggered := false

	das.stateChange <- status.ConnectionState

	for {
		das.UpdateAgentStatus(status)

		select {
		case sig := <-signals:
			log.Infof("Received signal %s, exiting...", sig)
			return

		case <-versionCheckTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
			status.NewVersionAvailable, err = newVersionAvailable(ctx)
			cancel()

			if err != nil {
				log.Errorf("check for new version: %s", err)
				continue
			}

			if status.NewVersionAvailable {
				notify.Infof("New version of device agent available: https://doc.nais.io/device/install#installation")
				versionCheckTicker.Stop()
			} else {
				versionCheckTicker.Reset(versionCheckInterval)
			}

		case <-certRenewalTicker.C:
			certRenewalTicker.Reset(certRenewalBackoff)
			if status.ConnectionState == pb.AgentState_Connected {
				err := clientcert.Renew()
				if err != nil {
					log.Errorf("Renewing NAV microsoft client certificate: %v", err)
					break
				}

				certRenewalTicker.Reset(certRenewalInterval)
				log.Info("NAV Microsoft Client Certificate renewed")
			} else {
				log.Debugf("cert renewal skipped, not connected")
			}

		case <-healthCheckTicker.C:
			healthCheckTicker.Reset(healthCheckInterval)
			if status.ConnectionState != pb.AgentState_Connected {
				break
			}

			wg := &sync.WaitGroup{}

			total := len(status.GetGateways())
			log.Infof("Pinging %d gateways...", total)
			for i, gw := range status.GetGateways() {
				go func(i int, gw *pb.Gateway) {
					wg.Add(1)
					err := ping(gw.Ip)
					pos := fmt.Sprintf("[%02d/%02d]", i+1, total)
					if err == nil {
						gw.Healthy = true
						log.Debugf("%s Successfully pinged gateway %v with ip: %v", pos, gw.Name, gw.Ip)
					} else {
						gw.Healthy = false
						log.Infof("%s unable to ping host %s: %v", pos, gw.Ip, err)
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
					das.rc.BootstrapConfig.Gateway(),
				},
				status.GetGateways()...,
			))
			cancel()

			if err != nil {
				notify.Errorf(err.Error())
				das.stateChange <- pb.AgentState_Disconnecting
			} else {
				if status.ConnectionState == pb.AgentState_Connected {
					healthCheckTicker.Reset(1 * time.Second)
				}
			}

		case newState := <-das.stateChange:
			writeStatusTofile(filepath.Join(das.Config.ConfigDir, "agent_status"), newState)
			previousState := status.ConnectionState
			status.ConnectionState = newState
			log.Infof("state changed to %s", status.ConnectionState)

			switch status.ConnectionState {
			case pb.AgentState_Bootstrapping:
				if das.rc.BootstrapConfig != nil {
					log.Infof("Already bootstrapped")
				} else {
					ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
					das.rc.BootstrapConfig, err = runtimeconfig.EnsureBootstrapping(das.rc, ctx)
					cancel()
					if err != nil {
						notify.Errorf("Bootstrap: %v", err)
						das.stateChange <- pb.AgentState_Disconnecting
						continue
					}
				}

				ctx, cancel := context.WithTimeout(context.Background(), helperTimeout)
				err = das.ConfigureHelper(ctx, das.rc, []*pb.Gateway{
					das.rc.BootstrapConfig.Gateway(),
				})
				cancel()

				if err != nil {
					notify.Errorf(err.Error())
					das.stateChange <- pb.AgentState_Disconnecting
					continue
				}

				status.ConnectedSince = timestamppb.Now()

				syncctx, synccancel = context.WithCancel(context.Background())
				go func() {
					for syncctx.Err() == nil {
						log.Infof("Setting up gateway config synchronization loop")
						err := das.syncConfigLoop(syncctx, gateways)
						if err != nil {
							if grpcstatus.Code(err) == codes.Unauthenticated {
								notify.Errorf(err.Error())
								das.stateChange <- pb.AgentState_Disconnecting
								synccancel()
							}

							log.Errorf("Synchronize config: %s", err)
							time.Sleep(1 * time.Second)
						}
					}

					log.Infof("Gateway config synchronization loop: %s", syncctx.Err())
					syncctx = nil
					synccancel()
				}()

				das.stateChange <- pb.AgentState_Connected
				continue

			case pb.AgentState_Authenticating:
				err = validateToken(das.rc.Token)
				if err == nil {
					log.Infof("Already have valid Azure AD token")
				} else {
					log.Infof("validate token: %v", err)

					ctx, cancel := context.WithTimeout(context.Background(), authFlowTimeout)
					das.rc.Token, err = auth.GetDeviceAgentToken(ctx, das.rc.Config.OAuth2Config)
					cancel()
					if err != nil {
						notify.Errorf("Get Azure AD token: %v", err)
						das.stateChange <- pb.AgentState_Disconnected
						break
					}
				}

				das.stateChange <- pb.AgentState_Bootstrapping

			case pb.AgentState_Connected:
				certRenewalTicker.Reset(1 * time.Second)

			case pb.AgentState_Disconnected:
				status.Gateways = make([]*pb.Gateway, 0)
				if das.Config.AgentConfiguration.AutoConnect && !autoConnectTriggered {
					autoConnectTriggered = true
					das.stateChange <- pb.AgentState_Authenticating
				}

			case pb.AgentState_Quitting:
				return

			case pb.AgentState_Disconnecting:
				if synccancel != nil {
					synccancel() // cancel streaming gateway updates
				}
				das.rc.SessionInfo = nil
				log.Info("Tearing down network connections through device-helper...")
				ctx, cancel := context.WithTimeout(context.Background(), helperTimeout)
				_, err = das.DeviceHelper.Teardown(ctx, &pb.TeardownRequest{})
				cancel()

				if err != nil {
					notify.Errorf(err.Error())
				}

				das.stateChange <- pb.AgentState_Disconnected

			case pb.AgentState_Unhealthy:

			case pb.AgentState_AgentConfigurationChanged:
				if das.Config.AgentConfiguration.CertRenewal {
					certRenewalTicker.Reset(1 * time.Second)
				}
				das.stateChange <- previousState
			}
		}
	}
}

func writeStatusTofile(path string, state pb.AgentState) {
	err := ioutil.WriteFile(path, []byte(state.String()), 0644)
	if err != nil {
		log.Errorf("unable to write agent status to file: %v", err)
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

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Errorf("close request body: %v", err)
		}
	}()
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
