package database

import (
	"context"
	"time"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/nais/device/internal/apiserver/sqlc"
	"github.com/nais/device/pkg/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Database interface {
	ReadDevices(ctx context.Context) ([]*pb.Device, error)
	ReadPeers(ctx context.Context) ([]*peer, error)
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
	SetDeviceSeenByKolide(ctx context.Context, externalID, serial, platform string, lastSeen *time.Time) error
	UpdateKolideIssues(ctx context.Context, issues []*kolide.DeviceFailure) error
	UpdateKolideIssuesForDevice(ctx context.Context, externalID string, issues []*kolide.DeviceFailure) error
	UpdateKolideChecks(ctx context.Context, checks []*kolide.Check) error
	ReadKolideChecks(ctx context.Context) (map[int64]*sqlc.KolideCheck, error)
	AcceptAcceptableUse(ctx context.Context, userID string) error
	RejectAcceptableUse(ctx context.Context, userID string) error
	GetAcceptances(ctx context.Context) (map[string]struct{}, error)
	GetAcceptedAt(ctx context.Context, userID string) (*timestamppb.Timestamp, error)
}
