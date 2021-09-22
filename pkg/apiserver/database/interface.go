package database

import (
	"context"

	"github.com/nais/device/pkg/pb"
)

type APIServer interface {
	ReadDevices() ([]*pb.Device, error)
	UpdateDevices(ctx context.Context, devices []*pb.Device) error
	UpdateGateway(ctx context.Context, name string, routes, accessGroupIDs []string, requiresPrivilegedAccess bool) error
	AddGateway(ctx context.Context, name, endpoint, publicKey string) error
	AddDevice(ctx context.Context, device *pb.Device) error
	ReadDevice(publicKey string) (*pb.Device, error)
	ReadDeviceById(ctx context.Context, deviceID int64) (*pb.Device, error)
	ReadGateways() ([]*pb.Gateway, error)
	ReadGateway(name string) (*pb.Gateway, error)
	ReadDeviceBySerialPlatform(ctx context.Context, serial string, platform string) (*pb.Device, error)
	AddSessionInfo(ctx context.Context, si *pb.Session) error
	ReadSessionInfo(ctx context.Context, key string) (*pb.Session, error)
	ReadSessionInfos(ctx context.Context) ([]*pb.Session, error)
	ReadMostRecentSessionInfo(ctx context.Context, deviceID int64) (*pb.Session, error)
	Migrate(ctx context.Context) error
}
