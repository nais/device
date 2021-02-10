// device-helper is responsible for:
// - running the WireGuard process
// - configuring the network tunnel interface
// - synchronizing WireGuard with the provided config
// - setting up the required routes
package device_helper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/nais/device/device-agent/wireguard"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	syncConfigWait = 2 * time.Second // wg syncconf is non-blocking; allow this much time for changes to propagate
)

type OSConfigurator interface {
	SetupInterface(ctx context.Context, cfg *pb.Configuration) error
	TeardownInterface(ctx context.Context) error
	SyncConf(ctx context.Context, cfg *pb.Configuration) error
	SetupRoutes(ctx context.Context, gateways []*pb.Gateway) error
	Prerequisites() error
}

type DeviceHelperServer struct {
	pb.UnimplementedDeviceHelperServer
	Config         Config
	OSConfigurator OSConfigurator
}

func (dhs *DeviceHelperServer) Configure(server pb.DeviceHelper_ConfigureServer) error {
	// fixme: locking/singleton

	defer func() {
		log.Infof("Removing network interface '%s' and all routes", dhs.Config.Interface)
		err := dhs.OSConfigurator.TeardownInterface(context.Background())
		if err != nil {
			log.Errorf("Tearing down interface: %v", err)
		}

		log.Infof("Flushing WireGuard configuration from disk")
		err = os.Remove(dhs.Config.WireGuardConfigPath)
		if err != nil {
			log.Error(err)
		}
	}()

	for {
		cfg, err := server.Recv()
		if err != nil {
			return err
		}

		log.Infof("New configuration received from device-agent")

		err = dhs.writeConfigFile(cfg)
		if err != nil {
			return status.Errorf(codes.ResourceExhausted, "write WireGuard configuration: %s", err)
		}

		log.Infof("Wrote WireGuard config to disk")

		err = dhs.OSConfigurator.SetupInterface(server.Context(), cfg)
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "setup interface and routes: %s", err)
		}

		err = dhs.OSConfigurator.SyncConf(server.Context(), cfg)
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "synchronize WireGuard configuration: %s", err)
		}

		err = dhs.OSConfigurator.SetupRoutes(server.Context(), cfg.GetGateways())
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "setting up routes: %s", err)
		}

		time.Sleep(syncConfigWait)
	}
}

func (dhs *DeviceHelperServer) writeConfigFile(cfg *pb.Configuration) error {
	buf := new(bytes.Buffer)

	_, err := wireguard.Marshal(buf, cfg)
	if err != nil {
		return fmt.Errorf("render configuration: %s", err)
	}

	fd, err := os.OpenFile(dhs.Config.WireGuardConfigPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("open file: %s", err)
	}
	defer fd.Close()

	_, err = io.Copy(fd, buf)
	if err != nil {
		return fmt.Errorf("write to disk: %s", err)
	}

	return nil
}

func (dhs *DeviceHelperServer) Upgrade(context.Context, *pb.UpgradeRequest) (*pb.UpgradeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Upgrade not implemented")
}
