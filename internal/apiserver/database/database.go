package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/internal/apiserver/ip"
	"github.com/nais/device/internal/apiserver/sqlc"
	"github.com/nais/device/internal/pb"
)

type ApiServerDB struct {
	queries             Querier
	ipv4Allocator       ip.Allocator
	ipv6Allocator       ip.Allocator
	defaultDeviceHealth bool
}

var mux sync.Mutex

func New(_ context.Context, dbPath string, v4Allocator ip.Allocator, v6Allocator ip.Allocator, defaultDeviceHealth bool) (APIServer, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	apiServerDB := ApiServerDB{
		queries:             NewQuerier(db),
		ipv4Allocator:       v4Allocator,
		ipv6Allocator:       v6Allocator,
		defaultDeviceHealth: defaultDeviceHealth,
	}

	if err = runMigrations(dbPath); err != nil {
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
				Healthy:  device.Healthy,
				Serial:   device.Serial,
				Platform: device.Platform,
				LastUpdated: sql.NullString{
					String: timeToString(time.Now().UTC()),
					Valid:  true,
				},
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
	err := db.queries.Transaction(ctx, func(ctx context.Context, qtx *sqlc.Queries) error {
		err := qtx.UpdateGateway(ctx, sqlc.UpdateGatewayParams{
			PublicKey:                gw.PublicKey,
			Endpoint:                 gw.Endpoint,
			Ipv4:                     gw.Ipv4,
			Ipv6:                     gw.Ipv6,
			RequiresPrivilegedAccess: gw.RequiresPrivilegedAccess,
			PasswordHash:             gw.PasswordHash,
			Name:                     gw.Name,
		})
		if err != nil {
			return err
		}

		err = qtx.DeleteGatewayAccessGroupIDs(ctx, gw.Name)
		if err != nil {
			return err
		}

		err = qtx.DeleteGatewayRoutes(ctx, gw.Name)
		if err != nil {
			return err
		}

		for _, groupID := range gw.AccessGroupIDs {
			err = qtx.AddGatewayAccessGroupID(ctx, sqlc.AddGatewayAccessGroupIDParams{
				GatewayName: gw.Name,
				GroupID:     groupID,
			})
			if err != nil {
				return err
			}
		}

		for _, route := range gw.GetRoutesIPv6() {
			err = qtx.AddGatewayRoute(ctx, sqlc.AddGatewayRouteParams{
				GatewayName: gw.Name,
				Route:       route,
				Family:      "IPv6",
			})
			if err != nil {
				return err
			}
		}
		for _, route := range gw.GetRoutesIPv4() {
			err = qtx.AddGatewayRoute(ctx, sqlc.AddGatewayRouteParams{
				GatewayName: gw.Name,
				Route:       route,
				Family:      "IPv4",
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("updating gateway: %w", err)
	}

	return nil
}

func (db *ApiServerDB) UpdateGatewayDynamicFields(ctx context.Context, gw *pb.Gateway) error {
	err := db.queries.Transaction(ctx, func(ctx context.Context, qtx *sqlc.Queries) error {
		err := qtx.UpdateGatewayDynamicFields(ctx, sqlc.UpdateGatewayDynamicFieldsParams{
			RequiresPrivilegedAccess: gw.RequiresPrivilegedAccess,
			Name:                     gw.Name,
		})
		if err != nil {
			return err
		}

		err = qtx.DeleteGatewayAccessGroupIDs(ctx, gw.Name)
		if err != nil {
			return err
		}

		err = qtx.DeleteGatewayRoutes(ctx, gw.Name)
		if err != nil {
			return err
		}

		for _, groupID := range gw.AccessGroupIDs {
			err = qtx.AddGatewayAccessGroupID(ctx, sqlc.AddGatewayAccessGroupIDParams{
				GatewayName: gw.Name,
				GroupID:     groupID,
			})
			if err != nil {
				return err
			}
		}

		for _, route := range gw.GetRoutesIPv4() {
			err = qtx.AddGatewayRoute(ctx, sqlc.AddGatewayRouteParams{
				GatewayName: gw.Name,
				Route:       route,
				Family:      "IPv4",
			})
			if err != nil {
				return err
			}
		}

		for _, route := range gw.GetRoutesIPv6() {
			err = qtx.AddGatewayRoute(ctx, sqlc.AddGatewayRouteParams{
				GatewayName: gw.Name,
				Route:       route,
				Family:      "IPv6",
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("updating gateway dynamic fields: %w", err)
	}

	return nil
}

func (db *ApiServerDB) AddGateway(ctx context.Context, gw *pb.Gateway) error {
	mux.Lock()
	defer mux.Unlock()

	availableIPv4, err := db.getNextAvailableIPv4(ctx)
	if err != nil {
		return fmt.Errorf("finding available ipv4: %w", err)
	}

	availableIPv6, err := db.getNextAvailableIPv6(ctx)
	if err != nil {
		return fmt.Errorf("finding available ipv6: %w", err)
	}

	err = db.queries.Transaction(ctx, func(ctx context.Context, qtx *sqlc.Queries) error {
		err = qtx.AddGateway(ctx, sqlc.AddGatewayParams{
			Name:                     gw.Name,
			Endpoint:                 gw.Endpoint,
			PublicKey:                gw.PublicKey,
			Ipv4:                     availableIPv4,
			Ipv6:                     availableIPv6,
			PasswordHash:             gw.PasswordHash,
			RequiresPrivilegedAccess: gw.RequiresPrivilegedAccess,
		})
		if err != nil {
			return err
		}

		for _, groupID := range gw.AccessGroupIDs {
			err = qtx.AddGatewayAccessGroupID(ctx, sqlc.AddGatewayAccessGroupIDParams{
				GatewayName: gw.Name,
				GroupID:     groupID,
			})
			if err != nil {
				return err
			}
		}

		for _, route := range gw.GetRoutesIPv4() {
			err = qtx.AddGatewayRoute(ctx, sqlc.AddGatewayRouteParams{
				GatewayName: gw.Name,
				Route:       route,
				Family:      "IPv4",
			})
			if err != nil {
				return err
			}
		}

		for _, route := range gw.GetRoutesIPv6() {
			err = qtx.AddGatewayRoute(ctx, sqlc.AddGatewayRouteParams{
				GatewayName: gw.Name,
				Route:       route,
				Family:      "IPv6",
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("inserting new gateway: %w", err)
	}

	return nil
}

func (db *ApiServerDB) AddDevice(ctx context.Context, device *pb.Device) error {
	mux.Lock()
	defer mux.Unlock()

	availableIpV4, err := db.getNextAvailableIPv4(ctx)
	if err != nil {
		return fmt.Errorf("finding available ip: %w", err)
	}

	availableIpV6, err := db.getNextAvailableIPv6(ctx)
	if err != nil {
		return fmt.Errorf("finding available ip: %w", err)
	}

	err = db.queries.AddDevice(ctx, sqlc.AddDeviceParams{
		Serial:    device.Serial,
		Username:  device.Username,
		PublicKey: device.PublicKey,
		Ipv4:      availableIpV4,
		Ipv6:      availableIpV6,
		Healthy:   db.defaultDeviceHealth,
		Platform:  device.Platform,
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
	device, err := db.queries.GetDeviceByID(ctx, deviceID)
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
		accessGroupIDs, err := db.queries.GetGatewayAccessGroupIDs(ctx, row.Name)
		if err != nil {
			return nil, err
		}

		routes, err := db.queries.GetGatewayRoutes(ctx, row.Name)
		if err != nil {
			return nil, err
		}

		gateways = append(gateways, sqlcGatewayToPbGateway(*row, accessGroupIDs, routes))
	}

	return gateways, nil
}

func (db *ApiServerDB) ReadGateway(ctx context.Context, name string) (*pb.Gateway, error) {
	gateway, err := db.queries.GetGatewayByName(ctx, name)
	if err != nil {
		return nil, err
	}

	accessGroupIDs, err := db.queries.GetGatewayAccessGroupIDs(ctx, name)
	if err != nil {
		return nil, err
	}

	routes, err := db.queries.GetGatewayRoutes(ctx, name)
	if err != nil {
		return nil, err
	}

	return sqlcGatewayToPbGateway(*gateway, accessGroupIDs, routes), nil
}

func (db *ApiServerDB) readExistingIPs(ctx context.Context) ([]string, error) {
	var ips []string

	devices, err := db.ReadDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading devices: %w", err)
	}

	for _, device := range devices {
		ips = append(ips, device.Ipv4)
	}

	gateways, err := db.ReadGateways(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading gateways: %w", err)
	}

	for _, gateway := range gateways {
		ips = append(ips, gateway.Ipv4)
	}

	return ips, nil
}

func (db *ApiServerDB) ReadDeviceBySerialPlatform(ctx context.Context, serial, platform string) (*pb.Device, error) {
	gateway, err := db.queries.GetDeviceBySerialAndPlatform(ctx, sqlc.GetDeviceBySerialAndPlatformParams{
		Serial:   serial,
		Platform: platform,
	})
	if err != nil {
		return nil, err
	}

	return sqlcDeviceToPbDevice(*gateway), nil
}

func (db *ApiServerDB) AddSessionInfo(ctx context.Context, si *pb.Session) error {
	err := db.queries.Transaction(ctx, func(ctx context.Context, qtx *sqlc.Queries) error {
		err := qtx.AddSession(ctx, sqlc.AddSessionParams{
			Key:      si.Key,
			Expiry:   timeToString(si.Expiry.AsTime().UTC()),
			DeviceID: si.GetDevice().GetId(),
			ObjectID: si.ObjectID,
		})
		if err != nil {
			return err
		}

		for _, groupID := range si.Groups {
			err = qtx.AddSessionAccessGroupID(ctx, sqlc.AddSessionAccessGroupIDParams{
				SessionKey: si.Key,
				GroupID:    groupID,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

func (db *ApiServerDB) ReadSessionInfo(ctx context.Context, key string) (*pb.Session, error) {
	row, err := db.queries.GetSessionByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	groupIDs, err := db.queries.GetSessionGroupIDs(ctx, key)
	if err != nil {
		return nil, err
	}

	return sqlcSessionAndDeviceToPbSession(row.Session, row.Device, groupIDs), nil
}

func (db *ApiServerDB) ReadSessionInfos(ctx context.Context) ([]*pb.Session, error) {
	rows, err := db.queries.GetSessions(ctx)
	if err != nil {
		return nil, err
	}

	sessions := make([]*pb.Session, 0)
	for _, row := range rows {
		groupIDs, err := db.queries.GetSessionGroupIDs(ctx, row.Session.Key)
		if err != nil {
			return nil, err
		}

		sessions = append(sessions, sqlcSessionAndDeviceToPbSession(row.Session, row.Device, groupIDs))
	}

	return sessions, nil
}

func (db *ApiServerDB) ReadMostRecentSessionInfo(ctx context.Context, deviceID int64) (*pb.Session, error) {
	row, err := db.queries.GetMostRecentDeviceSession(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	groupIDs, err := db.queries.GetSessionGroupIDs(ctx, row.Session.Key)
	if err != nil {
		return nil, err
	}

	return sqlcSessionAndDeviceToPbSession(row.Session, row.Device, groupIDs), nil
}

func (db *ApiServerDB) getNextAvailableIPv4(ctx context.Context) (string, error) {
	existingIps, err := db.readExistingIPs(ctx)
	if err != nil {
		return "", fmt.Errorf("reading existing ips: %w", err)
	}

	availableIp, err := db.ipv4Allocator.NextIP(existingIps)
	if err != nil {
		return "", fmt.Errorf("finding available ip: %w", err)
	}

	return availableIp, nil
}

func (db *ApiServerDB) getNextAvailableIPv6(ctx context.Context) (string, error) {
	lastUsedIP, err := db.queries.GetLastUsedIPV6(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "converting NULL to string is unsupported") {
			return db.ipv6Allocator.NextIP(nil)
		}
		return "", fmt.Errorf("getting last used ipv6: %w", err)
	}
	if lastUsedIP == "" {
		return db.ipv6Allocator.NextIP(nil)
	}

	return db.ipv6Allocator.NextIP([]string{lastUsedIP})
}

func (db *ApiServerDB) RemoveExpiredSessions(ctx context.Context) error {
	return db.queries.RemoveExpiredSessions(ctx)
}

func sqlcDeviceToPbDevice(sqlcDevice sqlc.Device) *pb.Device {
	pbDevice := &pb.Device{
		Id:        int64(sqlcDevice.ID),
		Serial:    sqlcDevice.Serial,
		Healthy:   sqlcDevice.Healthy,
		PublicKey: sqlcDevice.PublicKey,
		Ipv4:      sqlcDevice.Ipv4,
		Ipv6:      sqlcDevice.Ipv6,
		Username:  sqlcDevice.Username,
		Platform:  string(sqlcDevice.Platform),
	}

	if sqlcDevice.LastUpdated.Valid {
		pbDevice.LastUpdated = timestamppb.New(stringToTime(sqlcDevice.LastUpdated.String))
	}

	return pbDevice
}

func timeToString(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}

func stringToTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func sqlcGatewayToPbGateway(g sqlc.Gateway, groupIDs []string, routes []*sqlc.GetGatewayRoutesRow) *pb.Gateway {
	routesv4 := make([]string, 0)
	routesv6 := make([]string, 0)

	for _, route := range routes {
		switch route.Family {
		case "IPv4":
			routesv4 = append(routesv4, route.Route)
		case "IPv6":
			routesv6 = append(routesv6, route.Route)
		}
	}
	return &pb.Gateway{
		Name:                     g.Name,
		PublicKey:                g.PublicKey,
		Endpoint:                 g.Endpoint,
		Ipv4:                     g.Ipv4,
		Ipv6:                     g.Ipv6,
		RequiresPrivilegedAccess: g.RequiresPrivilegedAccess,
		PasswordHash:             g.PasswordHash,
		AccessGroupIDs:           groupIDs,
		RoutesIPv4:               routesv4,
		RoutesIPv6:               routesv6,
	}
}

func sqlcSessionAndDeviceToPbSession(s sqlc.Session, d sqlc.Device, groupIDs []string) *pb.Session {
	return &pb.Session{
		Key:      s.Key,
		Device:   sqlcDeviceToPbDevice(d),
		ObjectID: s.ObjectID,
		Expiry:   timestamppb.New(stringToTime(s.Expiry)),
		Groups:   groupIDs,
	}
}
