package device_agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/bootstrap"
	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/device-agent/wireguard"
	"github.com/nais/device/pkg/ioconvenience"
	"github.com/nais/device/pkg/pubsubenroll"

	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/device-agent/auth"
	"github.com/nais/device/pkg/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/notify"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/version"
)

const (
	healthCheckInterval    = 20 * time.Second   // how often to healthcheck gateways
	syncConfigDialTimeout  = 1 * time.Second    // sleep time between failed configuration syncs
	versionCheckInterval   = 1 * time.Hour      // how often to check for a new version of naisdevice
	versionCheckTimeout    = 3 * time.Second    // timeout for new version check
	getSerialTimeout       = 2 * time.Second    // timeout for getting device serial from helper
	authFlowTimeout        = 1 * time.Minute    // total timeout for authenticating user (AAD login in browser, redirect to localhost, exchange code for token)
	approximateInfinity    = time.Hour * 69_420 // Name describes purpose. Used for renewing microsoft client certs automatically
	certificateLifetime    = time.Hour * 23     // Microsoft Client certificate validity/renewal interval
	certCheckInterval      = time.Minute * 1    // Self-explanatory (Microsoft Client certificate)
	certRenewTimeout       = time.Second * 20
	apiServerRetryInterval = time.Second * 5
)

var (
	ErrExpiredToken      = errors.New("azure ad token expired")
	ErrTokenDoesNotExist = errors.New("azure ad token does not exist")
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
		return ErrTokenDoesNotExist
	} else if time.Now().After(token.Expiry) {
		return ErrExpiredToken
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
	if err != nil {
		return grpcstatus.Errorf(codes.Unavailable, err.Error())
	}

	log.Infof("Connected to API server")

	defer apiserver.Close()

	apiserverClient := pb.NewAPIServerClient(apiserver)

	if das.rc.SessionInfo.Expired() {
		serial, err := das.getSerial(ctx)
		if err != nil {
			return err
		}

		token := das.rc.Token.AccessToken
		if das.Config.EnableGoogleAuth {
			token = das.rc.Token.Extra("id_token").(string)
		}

		loginResponse, err := apiserverClient.Login(ctx, &pb.APIServerLoginRequest{
			Token:    token,
			Platform: config.Platform,
			Serial:   serial,
		})
		if err != nil {
			return err
		}

		das.rc.SessionInfo = loginResponse.Session
	}

	streamContext, cancel := context.WithDeadline(ctx, das.rc.SessionInfo.Expiry.AsTime())
	defer cancel()

	stream, err := apiserverClient.GetDeviceConfiguration(streamContext, &pb.GetDeviceConfigurationRequest{
		SessionKey: das.rc.SessionInfo.Key,
	})
	if err != nil {
		return err
	}

	log.Infof("Gateway configuration stream established")

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
			das.stateChange <- pb.AgentState_Disconnecting
			continue
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

func (das *DeviceAgentServer) EventLoop(ctx context.Context) {
	var err error
	var syncctx context.Context
	var synccancel context.CancelFunc

	gateways := make(chan []*pb.Gateway, 16)
	status := &pb.AgentStatus{}

	healthCheckTicker := time.NewTicker(healthCheckInterval)
	versionCheckTicker := time.NewTicker(5 * time.Second)
	certRenewalTicker := time.NewTicker(approximateInfinity)
	lastCertificateFetch := time.Time{}

	autoConnectTriggered := false

	das.stateChange <- status.ConnectionState

	for {
		das.UpdateAgentStatus(status)

		select {
		case <-ctx.Done():
			log.Infof("EventLoop: context done")
			return

		case <-versionCheckTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
			status.NewVersionAvailable, err = newVersionAvailable(ctx)
			cancel()

			if err != nil {
				log.Errorf("check for new version: %s", err)
				break
			}

			if status.NewVersionAvailable {
				notify.Infof("New version of device agent available: https://doc.nais.io/device/install#installation")
				versionCheckTicker.Stop()
			} else {
				versionCheckTicker.Reset(versionCheckInterval)
			}

		case <-certRenewalTicker.C:
			certRenewalTicker.Reset(certCheckInterval)

			if !das.Config.AgentConfiguration.CertRenewal {
				break
			}

			nextFetch := lastCertificateFetch.Add(certificateLifetime)
			if time.Now().Before(nextFetch) {
				break
			}

			if status.ConnectionState != pb.AgentState_Connected {
				log.Debugf("NAV Microsoft client certificate renewal not running because you are not connected")
				break
			}

			log.Infof("Attempting to install new NAV Microsoft client certificate")

			renewContext, renewCancel := context.WithTimeout(ctx, certRenewTimeout)
			err = das.outtune.Install(renewContext)
			renewCancel()

			if err != nil {
				log.Errorf("Renewing NAV Microsoft client certificate: %v", err)
				break
			}

			lastCertificateFetch = time.Now()

			log.Info("NAV Microsoft client certificate renewed")

		case <-healthCheckTicker.C:
			healthCheckTicker.Reset(healthCheckInterval)
			if status.ConnectionState != pb.AgentState_Connected {
				break
			}

			wg := &sync.WaitGroup{}

			total := len(status.GetGateways())
			log.Infof("Pinging %d gateways...", total)
			for i, gw := range status.GetGateways() {
				wg.Add(1)
				go func(i int, gw *pb.Gateway) {
					err := ping(gw.Ip)
					pos := fmt.Sprintf("[%02d/%02d]", i+1, total)
					if err == nil {
						gw.Healthy = true
						log.Debugf("%s %s: successfully pinged %v", pos, gw.Name, gw.Ip)
					} else {
						gw.Healthy = false
						log.Infof("%s %s: unable to ping %s: %v", pos, gw.Name, gw.Ip, err)
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
				das.stateChange <- pb.AgentState_Connected
			}

		case newState := <-das.stateChange:
			sentry.AddBreadcrumb(&sentry.Breadcrumb{
				Level:   sentry.LevelInfo,
				Message: "state changed",
				Type:    "debug",
				Data: map[string]any{
					"newState": newState.String(),
				},
				Category: "eventloop",
			})

			previousState := status.ConnectionState
			status.ConnectionState = newState
			log.Infof("state changed to %s", status.ConnectionState)

			switch status.ConnectionState {
			case pb.AgentState_Bootstrapping:
				if das.rc.BootstrapConfig != nil {
					log.Infof("Already bootstrapped")
				} else {
					ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
					serial, err := das.getSerial(ctx)
					if err != nil {
						notify.Errorf("Unable to get serial number: %v", err)
						das.stateChange <- pb.AgentState_Disconnecting
						cancel()
						continue
					}

					if das.Config.EnableGoogleAuth {
						req := &pubsubenroll.DeviceRequest{
							Platform:           das.Config.Platform,
							Serial:             serial,
							WireGuardPublicKey: wireguard.PublicKey(das.rc.PrivateKey),
						}
						var resp *pubsubenroll.Response
						resp, err = pubsubenroll.Enroll(ctx, req, das.rc.Token, das.Config.EnrollProjectID, das.Config.EnrollTopicName, log.WithField("component", "enroll"))
						if err == nil {
							das.rc.BootstrapConfig = &bootstrap.Config{
								DeviceIP:       resp.WireGuardIP,
								PublicKey:      resp.Peers[0].PublicKey,
								TunnelEndpoint: resp.Peers[0].Endpoint,
								APIServerIP:    strings.SplitN(resp.APIServerGRPCAddress, ":", 2)[0],
							}
						}
					} else {
						das.rc.BootstrapConfig, err = runtimeconfig.EnsureBootstrapping(das.rc, serial, ctx)
					}

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
					attempt := 0
					for syncctx.Err() == nil {
						attempt++
						log.Infof("[attempt %d] Setting up gateway configuration stream...", attempt)
						err := das.syncConfigLoop(syncctx, gateways)

						switch grpcstatus.Code(err) {
						case codes.OK:
							attempt = 0
						case codes.Unavailable:
							log.Warnf("Synchronize config: not connected to API server!")
							time.Sleep(apiServerRetryInterval)
						case codes.Unauthenticated:
							log.Errorf("Logging in: %s", err)
							das.rc.Token = nil
							log.Error("Cleaned up Azure AD Token")
							fallthrough
						default:
							notify.Errorf(err.Error())
							das.stateChange <- pb.AgentState_Disconnecting
							synccancel()
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

					ctx, cancel := context.WithTimeout(ctx, authFlowTimeout)
					das.rc.Token, err = auth.GetDeviceAgentToken(ctx, das.rc.Config.OAuth2Config, das.Config.GoogleAuthServerAddress)
					cancel()
					if err != nil {
						notify.Errorf("Get token: %v", err)
						das.stateChange <- pb.AgentState_Disconnected
						break
					}
				}

				das.stateChange <- pb.AgentState_Bootstrapping

			case pb.AgentState_Connected:
				sentry.CaptureMessage("Connected")
				healthCheckTicker.Reset(1 * time.Second)
				certRenewalTicker.Reset(5 * time.Second)

			case pb.AgentState_Disconnected:
				status.Gateways = make([]*pb.Gateway, 0)
				if das.Config.AgentConfiguration.AutoConnect && !autoConnectTriggered {
					autoConnectTriggered = true
					das.stateChange <- pb.AgentState_Authenticating
				}

			case pb.AgentState_Quitting:
				return

			case pb.AgentState_Disconnecting:
				sentry.CaptureMessage("Disconnected")
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
				certRenewalTicker.Reset(1 * time.Second)
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
