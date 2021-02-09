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

	"github.com/nais/device/device-agent/wireguard"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeviceHelperServer struct {
	pb.UnimplementedDeviceHelperServer
	cfg Config
}

func (dhs *DeviceHelperServer) Configure(server pb.DeviceHelper_ConfigureServer) error {
	// fixme: locking/singleton

	defer TeardownInterface(context.Background(), dhs.cfg.Interface)

	for {
		cfg, err := server.Recv()
		if err != nil {
			return err
		}

		log.Info("WireGuard configuration received")

		err = dhs.writeConfigFile(cfg)
		if err != nil {
			return status.Errorf(codes.ResourceExhausted, "write WireGuard configuration: %s", err)
		}

		log.Infof("Wrote WireGuard config to disk")

		err = setupInterface(server.Context(), dhs.cfg.Interface, cfg)
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "setup interface and routes: %s", err)
		}

		err = syncConf(server.Context(), dhs.cfg)
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "synchronize WireGuard configuration: %s", err)
		}

		err = setupRoutes(server.Context(), cfg.GetGateways(), dhs.cfg.Interface)
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "setting up routes: %s", err)
		}
	}
}

func (dhs *DeviceHelperServer) writeConfigFile(cfg *pb.Configuration) error {
	buf := new(bytes.Buffer)

	_, err := wireguard.Marshal(buf, cfg)
	if err != nil {
		return fmt.Errorf("render configuration: %s", err)
	}

	fd, err := os.OpenFile(dhs.cfg.WireGuardConfigPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
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
