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
	"time"

	"github.com/nais/device/internal/helper/serial"
	"github.com/nais/device/internal/ioconvenience"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nais/device/internal/deviceagent/wireguard"
	"github.com/nais/device/pkg/pb"
)

type OSConfigurator interface {
	SetupInterface(ctx context.Context, cfg *pb.Configuration) error
	TeardownInterface(ctx context.Context) error
	SyncConf(ctx context.Context, cfg *pb.Configuration) error
	SetupRoutes(ctx context.Context, gateways []*pb.Gateway) (routesAdded int, err error)
	Prerequisites() error
}

type DeviceHelperServer struct {
	pb.UnimplementedDeviceHelperServer
	config         Config
	osConfigurator OSConfigurator
	log            *logrus.Entry
}

func NewDeviceHelperServer(
	log *logrus.Entry,
	config Config,
	osConfigurator OSConfigurator,
) *DeviceHelperServer {
	return &DeviceHelperServer{
		log:            log,
		config:         config,
		osConfigurator: osConfigurator,
	}
}

func (dhs *DeviceHelperServer) Teardown(
	ctx context.Context,
	req *pb.TeardownRequest,
) (*pb.TeardownResponse, error) {
	dhs.log.WithField("interface", dhs.config.Interface).Info("removing network interface and all routes")
	err := dhs.osConfigurator.TeardownInterface(ctx)
	if err != nil {
		return nil, fmt.Errorf("tearing down interface: %v", err)
	}

	dhs.log.Info("flushing WireGuard configuration from disk")
	err = os.Remove(dhs.config.WireGuardConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("flush WireGuard configuration from disk: %v", err)
		}
		dhs.log.Info("WireGuard configuration file does not exist on disk")
	}

	return &pb.TeardownResponse{}, nil
}

func (dhs *DeviceHelperServer) Configure(
	ctx context.Context,
	cfg *pb.Configuration,
) (*pb.ConfigureResponse, error) {
	dhs.log.Info("new configuration received from device-agent")

	err := dhs.writeConfigFile(cfg)
	if err != nil {
		return nil, status.Errorf(codes.ResourceExhausted, "write WireGuard configuration: %s", err)
	}

	dhs.log.Info("wrote WireGuard config to disk")

	err = dhs.osConfigurator.SetupInterface(ctx, cfg)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "setup interface and routes: %s", err)
	}

	var loopErr error
	for attempt := range 5 {
		loopErr = dhs.osConfigurator.SyncConf(ctx, cfg)
		if loopErr != nil {
			backoff := time.Duration(attempt) * time.Second
			dhs.log.WithError(err).Error("synchronize WireGuard configuration")
			dhs.log.WithField("attempt", attempt+1).WithField("backoff", backoff).Info("configuring failed, sleeping before retrying")
			time.Sleep(backoff)
			continue
		}
		break
	}
	if loopErr != nil {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"synchronize WireGuard configuration: %s",
			loopErr,
		)
	}

	_, err = dhs.osConfigurator.SetupRoutes(ctx, cfg.GetGateways())
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

	fd, err := os.OpenFile(dhs.config.WireGuardConfigPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open file: %s", err)
	}
	defer ioconvenience.CloseWithLog(fd, dhs.log)

	_, err = io.Copy(fd, buf)
	if err != nil {
		return fmt.Errorf("write to disk: %s", err)
	}

	if err := fd.Sync(); err != nil {
		return fmt.Errorf("sync file: %s", err)
	}

	return nil
}

func (dhs *DeviceHelperServer) GetSerial(
	context.Context,
	*pb.GetSerialRequest,
) (*pb.GetSerialResponse, error) {
	deviceSerial, err := serial.GetDeviceSerial()
	if err != nil {
		return nil, err
	}
	return &pb.GetSerialResponse{Serial: deviceSerial}, nil
}

func (dhs *DeviceHelperServer) Ping(
	context.Context,
	*pb.PingRequest,
) (*pb.PingResponse, error) {
	return &pb.PingResponse{}, nil
}

func (dhs *DeviceHelperServer) Upgrade(
	context.Context,
	*pb.UpgradeRequest,
) (*pb.UpgradeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Upgrade not implemented")
}
