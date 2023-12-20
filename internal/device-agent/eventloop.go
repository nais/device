package device_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/version"
	"github.com/sirupsen/logrus"
)

const (
	healthCheckInterval    = 20 * time.Second // how often to healthcheck gateways
	syncConfigDialTimeout  = 1 * time.Second  // sleep time between failed configuration syncs
	versionCheckInterval   = 1 * time.Hour    // how often to check for a new version of naisdevice
	versionCheckTimeout    = 3 * time.Second  // timeout for new version check
	getSerialTimeout       = 2 * time.Second  // timeout for getting device serial from helper
	authFlowTimeout        = 1 * time.Minute  // total timeout for authenticating user (AAD login in browser, redirect to localhost, exchange code for token)
	apiServerRetryInterval = time.Millisecond * 10
)

func (das *DeviceAgentServer) Run(ctx context.Context, rc runtimeconfig.RuntimeConfig, cfg config.Config, helper pb.DeviceHelperClient, notifier notify.Notifier) {
}

func (das *DeviceAgentServer) EventLoop(programContext context.Context) {
	// case <-versionCheckTicker.C:
	// 	ctx, cancel := context.WithTimeout(programContext, versionCheckTimeout)
	// 	status.NewVersionAvailable, err = newVersionAvailable(ctx)
	// 	cancel()
	//
	// 	if err != nil {
	// 		das.log.Errorf("check for new version: %s", err)
	// 		break
	// 	}
	//
	// 	if status.NewVersionAvailable {
	// 		das.Notifier().Infof("New version of device agent available: https://doc.nais.io/device/update/")
	// 		versionCheckTicker.Stop()
	// 	} else {
	// 		versionCheckTicker.Reset(versionCheckInterval)
	// 	}
	//
	// case <-healthCheckTicker.C:
	// 	healthCheckTicker.Reset(healthCheckInterval)
	// 	if status.ConnectionState != pb.AgentState_Connected {
	// 		break
	// 	}
	//
	// 	helperHealthCheckCtx, cancel := context.WithTimeout(programContext, 5*time.Second)
	// 	if _, err := das.DeviceHelper.GetSerial(helperHealthCheckCtx, &pb.GetSerialRequest{}); err != nil {
	// 		cancel()
	//
	// 		das.log.WithError(err).Errorf("Unable to communicate with helper. Shutting down")
	// 		das.notifier.Errorf("Unable to communicate with helper. Shutting down.")
	//
	// 		das.stateChange <- pb.AgentState_Disconnecting
	// 		break
	// 	}
	// 	cancel()
	//
	// 	wg := &sync.WaitGroup{}
	//
	// 	total := len(status.GetGateways())
	// 	das.log.Debugf("Pinging %d gateways...", total)
	// 	for i, gw := range status.GetGateways() {
	// 		wg.Add(1)
	// 		go func(i int, gw *pb.Gateway) {
	// 			err := ping(das.log, gw.Ipv4)
	// 			pos := fmt.Sprintf("[%02d/%02d]", i+1, total)
	// 			if err == nil {
	// 				gw.Healthy = true
	// 				das.log.Debugf("%s %s: successfully pinged %v", pos, gw.Name, gw.Ipv4)
	// 			} else {
	// 				gw.Healthy = false
	// 				das.log.Debugf("%s %s: unable to ping %s: %v", pos, gw.Name, gw.Ipv4, err)
	// 			}
	// 			wg.Done()
	// 		}(i, gw)
	// 	}
	// 	wg.Wait()
}

func newVersionAvailable(ctx context.Context) (bool, error) {
	type response struct {
		Tag string `json:"tag_name"`
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/nais/device/releases/latest", nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("retrieve current release version: %s", err)
	}

	defer resp.Body.Close()

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

func ping(log *logrus.Entry, addr string) error {
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
