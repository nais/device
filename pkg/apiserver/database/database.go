package database

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/jackc/pgx/v4"
	"github.com/nais/device/pkg/apiserver/sqlc"
	"github.com/nais/device/pkg/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ApiServerDB struct {
	queries             Querier
	IPAllocator         IPAllocator
	defaultDeviceHealth bool
}

var mux sync.Mutex

func New(ctx context.Context, dsn string, ipAllocator IPAllocator, defaultDeviceHealth bool) (APIServer, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}

	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	apiServerDB := ApiServerDB{
		queries:             NewQuerier(conn),
		IPAllocator:         ipAllocator,
		defaultDeviceHealth: defaultDeviceHealth,
	}

	if err = runMigrations(cfg.ConnString()); err != nil {
		return nil, fmt.Errorf("migrating database: %w", err)
	}

	return &apiServerDB, nil
}

func (db *ApiServerDB) ReadDevices(ctx context.Context) ([]*pb.Device, error) {
	rows, err := db.queries.GetDevices(ctx)
	if err != nil {
		return nil, err
	}

	devices := make([]*pb.Device, 0)
	for _, row := range rows {
		devices = append(devices, sqlcDeviceToPbDevice(*row))
	}

	return devices, nil
}

func (db *ApiServerDB) UpdateDevices(ctx context.Context, devices []*pb.Device) error {
	err := db.queries.Transaction(ctx, func(ctx context.Context, queries *sqlc.Queries) error {
		for _, device := range devices {
			err := queries.UpdateDevice(ctx, sqlc.UpdateDeviceParams{
				Healthy:  &device.Healthy,
				Serial:   &device.Serial,
				Platform: sqlc.Platform(device.Platform),
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error in transaction: %w", err)
	}

	return nil
}

func (db *ApiServerDB) UpdateGateway(ctx context.Context, gw *pb.Gateway) error {
	accessGroupIDs := strings.Join(gw.AccessGroupIDs, ",")
	routes := strings.Join(gw.Routes, ",")

	err := db.queries.UpdateGateway(ctx, sqlc.UpdateGatewayParams{
		PublicKey:                gw.PublicKey,
		AccessGroupIds:           &accessGroupIDs,
		Endpoint:                 &gw.Endpoint,
		Ip:                       &gw.Ip,
		Routes:                   &routes,
		RequiresPrivilegedAccess: &gw.RequiresPrivilegedAccess,
		PasswordHash:             &gw.PasswordHash,
		Name:                     gw.Name,
	})
	if err != nil {
		return fmt.Errorf("updating gateway: %w", err)
	}

	return nil
}

func (db *ApiServerDB) UpdateGatewayDynamicFields(ctx context.Context, gw *pb.Gateway) error {
	accessGroupIDs := strings.Join(gw.AccessGroupIDs, ",")
	routes := strings.Join(gw.Routes, ",")

	err := db.queries.UpdateGatewayDynamicFields(ctx, sqlc.UpdateGatewayDynamicFieldsParams{
		AccessGroupIds:           &accessGroupIDs,
		Routes:                   &routes,
		RequiresPrivilegedAccess: &gw.RequiresPrivilegedAccess,
		Name:                     gw.Name,
	})
	if err != nil {
		return fmt.Errorf("updating gateway dynamic fields: %w", err)
	}

	return nil
}

func (db *ApiServerDB) AddGateway(ctx context.Context, gw *pb.Gateway) error {
	mux.Lock()
	defer mux.Unlock()

	availableIp, err := db.getNextAvailableIp(ctx)
	if err != nil {
		return fmt.Errorf("finding available ip: %w", err)
	}

	accessGroupIDs := strings.Join(gw.AccessGroupIDs, ",")
	routes := strings.Join(gw.Routes, ",")

	err = db.queries.AddGateway(ctx, sqlc.AddGatewayParams{
		Name:                     gw.Name,
		Endpoint:                 &gw.Endpoint,
		PublicKey:                gw.PublicKey,
		Ip:                       availableIp,
		PasswordHash:             &gw.PasswordHash,
		AccessGroupIds:           &accessGroupIDs,
		Routes:                   &routes,
		RequiresPrivilegedAccess: &gw.RequiresPrivilegedAccess,
	})
	if err != nil {
		return fmt.Errorf("inserting new gateway: %w", err)
	}

	return nil
}

func (db *ApiServerDB) AddDevice(ctx context.Context, device *pb.Device) error {
	mux.Lock()
	defer mux.Unlock()

	availableIp, err := db.getNextAvailableIp(ctx)
	if err != nil {
		return fmt.Errorf("finding available ip: %w", err)
	}

	err = db.queries.AddDevice(ctx, sqlc.AddDeviceParams{
		Serial:    &device.Serial,
		Username:  &device.Username,
		PublicKey: device.PublicKey,
		Ip:        availableIp,
		Healthy:   &db.defaultDeviceHealth,
		Platform:  sqlc.Platform(device.Platform),
	})
	if err != nil {
		return fmt.Errorf("inserting new device: %w", err)
	}

	return nil
}

func (db *ApiServerDB) ReadDevice(ctx context.Context, publicKey string) (*pb.Device, error) {
	device, err := db.queries.GetDeviceByPublicKey(ctx, publicKey)
	if err != nil {
		return nil, err
	}

	return sqlcDeviceToPbDevice(*device), nil
}

func (db *ApiServerDB) ReadDeviceById(ctx context.Context, deviceID int64) (*pb.Device, error) {
	device, err := db.queries.GetDeviceByID(ctx, int32(deviceID))
	if err != nil {
		return nil, err
	}

	return sqlcDeviceToPbDevice(*device), nil
}

func (db *ApiServerDB) ReadGateways(ctx context.Context) ([]*pb.Gateway, error) {
	rows, err := db.queries.GetGateways(ctx)
	if err != nil {
		return nil, err
	}

	gateways := make([]*pb.Gateway, 0)
	for _, row := range rows {
		gateway := sqlcGatewayToPbGateway(*row)
		gateways = append(gateways, gateway)
	}

	return gateways, nil
}

func (db *ApiServerDB) ReadGateway(ctx context.Context, name string) (*pb.Gateway, error) {
	gateway, err := db.queries.GetGatewayByName(ctx, name)
	if err != nil {
		return nil, err
	}

	return sqlcGatewayToPbGateway(*gateway), nil
}

func (db *ApiServerDB) readExistingIPs(ctx context.Context) ([]string, error) {
	var ips []string

	devices, err := db.ReadDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading devices: %w", err)
	}

	for _, device := range devices {
		ips = append(ips, device.Ip)
	}

	gateways, err := db.ReadGateways(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading gateways: %w", err)
	}

	for _, gateway := range gateways {
		ips = append(ips, gateway.Ip)
	}

	return ips, nil
}

func (db *ApiServerDB) ReadDeviceBySerialPlatform(ctx context.Context, serial, platform string) (*pb.Device, error) {
	gateway, err := db.queries.GetDeviceBySerialAndPlatform(ctx, sqlc.GetDeviceBySerialAndPlatformParams{
		Serial:   &serial,
		Platform: sqlc.Platform(platform),
	})
	if err != nil {
		return nil, err
	}

	return sqlcDeviceToPbDevice(*gateway), nil
}

func (db *ApiServerDB) AddSessionInfo(ctx context.Context, si *pb.Session) error {
	expiry := si.Expiry.AsTime()
	groups := strings.Join(si.Groups, ",")

	err := db.queries.AddSession(ctx, sqlc.AddSessionParams{
		Key:      &si.Key,
		Expiry:   &expiry,
		DeviceID: int32(si.GetDevice().GetId()),
		Groups:   &groups,
		ObjectID: &si.ObjectID,
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

func (db *ApiServerDB) ReadSessionInfo(ctx context.Context, key string) (*pb.Session, error) {
	row, err := db.queries.GetSessionByKey(ctx, &key)
	if err != nil {
		return nil, err
	}

	return sqlcSessionAndDeviceToPbSession(row.Session, row.Device), nil
}

func (db *ApiServerDB) ReadSessionInfos(ctx context.Context) ([]*pb.Session, error) {
	rows, err := db.queries.GetSessions(ctx)
	if err != nil {
		return nil, err
	}

	sessions := make([]*pb.Session, 0)
	for _, row := range rows {
		sessions = append(sessions, sqlcSessionAndDeviceToPbSession(row.Session, row.Device))
	}

	return sessions, nil
}

func (db *ApiServerDB) ReadMostRecentSessionInfo(ctx context.Context, deviceID int64) (*pb.Session, error) {
	row, err := db.queries.GetMostRecentDeviceSession(ctx, int32(deviceID))
	if err != nil {
		return nil, err
	}

	return sqlcSessionAndDeviceToPbSession(row.Session, row.Device), nil
}

func (db *ApiServerDB) getNextAvailableIp(ctx context.Context) (*string, error) {
	existingIps, err := db.readExistingIPs(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading existing ips: %w", err)
	}

	availableIp, err := db.IPAllocator.NextIP(existingIps)
	if err != nil {
		return nil, fmt.Errorf("finding available ip: %w", err)
	}

	return &availableIp, nil
}

func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func derefBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

func sqlcDeviceToPbDevice(d sqlc.Device) *pb.Device {
	device := &pb.Device{
		Id:        int64(d.ID),
		Serial:    derefString(d.Serial),
		Healthy:   derefBool(d.Healthy),
		PublicKey: d.PublicKey,
		Ip:        derefString(d.Ip),
		Username:  derefString(d.Username),
		Platform:  string(d.Platform),
	}

	if d.LastUpdated != nil {
		device.LastUpdated = timestamppb.New(*d.LastUpdated)
	}

	return device
}

func sqlcGatewayToPbGateway(g sqlc.Gateway) *pb.Gateway {
	gateway := &pb.Gateway{
		Name:                     g.Name,
		PublicKey:                g.PublicKey,
		Endpoint:                 derefString(g.Endpoint),
		Ip:                       derefString(g.Ip),
		RequiresPrivilegedAccess: derefBool(g.RequiresPrivilegedAccess),
	}

	if g.PasswordHash != nil {
		gateway.PasswordHash = *g.PasswordHash
	}

	if g.AccessGroupIds != nil && len(*g.AccessGroupIds) > 0 {
		gateway.AccessGroupIDs = strings.Split(*g.AccessGroupIds, ",")
	}

	if g.Routes != nil && len(*g.Routes) > 0 {
		gateway.Routes = strings.Split(*g.Routes, ",")
	}

	return gateway
}

func sqlcSessionAndDeviceToPbSession(s sqlc.Session, d sqlc.Device) *pb.Session {
	session := &pb.Session{
		Key:      derefString(s.Key),
		Device:   sqlcDeviceToPbDevice(d),
		ObjectID: derefString(s.ObjectID),
	}

	if s.Expiry != nil {
		session.Expiry = timestamppb.New(*s.Expiry)
	}

	if s.Groups != nil && len(*s.Groups) > 0 {
		session.Groups = strings.Split(*s.Groups, ",")
	}

	return session
}
