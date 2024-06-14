package database

import (
	"context"
	"time"

	"github.com/nais/device/internal/pb"
)

type APIServer interface {
	ReadDevices(ctx context.Context) ([]*pb.Device, error)
	UpdateDevices(ctx context.Context, devices []*pb.Device) error
	UpdateGateway(ctx context.Context, gateway *pb.Gateway) error
	UpdateGatewayDynamicFields(ctx context.Context, gateway *pb.Gateway) error
	AddGateway(ctx context.Context, gateway *pb.Gateway) error
	AddDevice(ctx context.Context, device *pb.Device) error
	ReadDevice(ctx context.Context, publicKey string) (*pb.Device, error)
	ReadDeviceById(ctx context.Context, deviceID int64) (*pb.Device, error)
	ReadDeviceByExternalID(ctx context.Context, externalID string) (*pb.Device, error)
	ReadGateways(ctx context.Context) ([]*pb.Gateway, error)
	ReadGateway(ctx context.Context, name string) (*pb.Gateway, error)
	ReadDeviceBySerialPlatform(ctx context.Context, serial string, platform string) (*pb.Device, error)
	AddSessionInfo(ctx context.Context, si *pb.Session) error
	ReadSessionInfo(ctx context.Context, key string) (*pb.Session, error)
	ReadSessionInfos(ctx context.Context) ([]*pb.Session, error)
	RemoveExpiredSessions(ctx context.Context) error
	ReadMostRecentSessionInfo(ctx context.Context, deviceID int64) (*pb.Session, error)
	ClearDeviceIssuesExceptFor(ctx context.Context, deviceIds []int64) error
	UpdateSingleDevice(ctx context.Context, externalID, serial, platform string, lastSeen *time.Time, issues []*pb.DeviceIssue) (int64, error)
}
