package device_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/version"
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
