// device-helper is responsible for:
// - running the WireGuard process
// - configuring the network tunnel interface
// - synchronizing WireGuard with the provided config
// - setting up the required routes
package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/nais/device/internal/helper/serial"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
		return nil, fmt.Errorf("tearing down interface: %w", err)
	}

	return &pb.TeardownResponse{}, nil
}

func (dhs *DeviceHelperServer) Configure(
	ctx context.Context,
	cfg *pb.Configuration,
) (*pb.ConfigureResponse, error) {
	dhs.log.Info("new configuration received from device-agent")

	err := dhs.osConfigurator.SetupInterface(ctx, cfg)
	if err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "setup interface and routes: %s", err)
	}

	var loopErr error
	for attempt := range 5 {
		loopErr = dhs.osConfigurator.SyncConf(ctx, cfg)
		if loopErr != nil {
			backoff := time.Duration(attempt) * time.Second
			dhs.log.WithError(loopErr).Error("synchronize WireGuard configuration")
			dhs.log.WithField("attempt", attempt+1).WithField("backoff", backoff).Info("configuring failed, sleeping before retrying")
			select {
			case <-ctx.Done():
				return nil, status.Errorf(codes.Canceled, "context canceled during WireGuard sync retry: %s", loopErr)
			case <-time.After(backoff):
			}
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
