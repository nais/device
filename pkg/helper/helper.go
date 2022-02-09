// device-helper is responsible for:
// - running the WireGuard process
// - configuring the network tunnel interface
// - synchronizing WireGuard with the provided config
// - setting up the required routes
package helper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/nais/device/pkg/helper/config"
	"github.com/nais/device/pkg/helper/serial"
	wireguard2 "github.com/nais/device/pkg/wireguard"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nais/device/pkg/device-agent/wireguard"
	"github.com/nais/device/pkg/pb"
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
	Wireguard      *wireguard2.Config
}

var (
	WireGuardConfigPath = filepath.Join(config.ConfigDir, "utun69.conf")
)

func (dhs *DeviceHelperServer) Teardown(ctx context.Context, req *pb.TeardownRequest) (*pb.TeardownResponse, error) {
	log.Infof("Removing network interface '%s' and all routes", dhs.Config.Interface)
	err := dhs.OSConfigurator.TeardownInterface(ctx)
	if err != nil {
		return nil, fmt.Errorf("tearing down interface: %v", err)
	}

	log.Infof("Flushing WireGuard configuration from disk")
	err = os.Remove(WireGuardConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("flush WireGuard configuration from disk: %v", err)
		}
		log.Infof("WireGuard configuration file does not exist on disk")
	}

	return &pb.TeardownResponse{}, nil
}

func (dhs *DeviceHelperServer) Configure(ctx context.Context, cfg *pb.Configuration) (*pb.ConfigureResponse, error) {
	log.Infof("New configuration received from device-agent")

	err := dhs.writeConfigFile(cfg)
	if err != nil {
		return nil, status.Errorf(codes.ResourceExhausted, "write WireGuard configuration: %s", err)
	}

	log.Infof("Wrote WireGuard config to disk")

	err = dhs.OSConfigurator.SetupInterface(ctx, cfg)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "setup interface and routes: %s", err)
	}

	err = dhs.OSConfigurator.SyncConf(ctx, cfg)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "synchronize WireGuard configuration: %s", err)
	}

	err = dhs.OSConfigurator.SetupRoutes(ctx, cfg.GetGateways())
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "setting up routes: %s", err)
	}

	return &pb.ConfigureResponse{}, nil
}

func (dhs *DeviceHelperServer) writeConfigFile(cfg *pb.Configuration) error {
	buf := new(bytes.Buffer)

	err := wireguard.Marshal(buf, cfg)
	if err != nil {
		return fmt.Errorf("render configuration: %s", err)
	}

	fd, err := os.OpenFile(WireGuardConfigPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
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

func (dhs *DeviceHelperServer) GetSerial(context.Context, *pb.GetSerialRequest) (*pb.GetSerialResponse, error) {
	device_serial, err := serial.GetDeviceSerial()
	if err != nil {
		return nil, err
	}
	return &pb.GetSerialResponse{Serial: device_serial}, nil
}

func (dhs *DeviceHelperServer) Upgrade(context.Context, *pb.UpgradeRequest) (*pb.UpgradeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Upgrade not implemented")
}
