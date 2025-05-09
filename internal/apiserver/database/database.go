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

	"github.com/sirupsen/logrus"

	"github.com/nais/device/internal/apiserver/ip"
	"github.com/nais/device/internal/apiserver/sqlc"
	"github.com/nais/device/pkg/pb"
)

type database struct {
	queries       Querier
	ipv4Allocator ip.Allocator
	ipv6Allocator ip.Allocator
	kolideEnabled bool
	log           logrus.FieldLogger
}

func (db *database) ReadKolideChecks(ctx context.Context) (map[int64]*sqlc.KolideCheck, error) {
	rows, err := db.queries.GetKolideChecks(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting kolide checks: %w", err)
	}

	checks := make(map[int64]*sqlc.KolideCheck)
	for _, row := range rows {
		checks[row.ID] = row
	}

	return checks, nil
}

var mux sync.Mutex

func New(dbPath string, v4Allocator ip.Allocator, v6Allocator ip.Allocator, kolideEnabled bool, log logrus.FieldLogger) (*database, error) {
	connectionString := "file:" + dbPath + "?_foreign_keys=1&_cache_size=-100000&_busy_timeout=5000&_journal_mode=WAL"
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	apiServerDB := database{
		queries:       NewQuerier(db),
		ipv4Allocator: v4Allocator,
		ipv6Allocator: v6Allocator,
		kolideEnabled: kolideEnabled,
		log:           log,
	}

	if err = runMigrations(dbPath); err != nil {
		return nil, fmt.Errorf("migrating database: %w", err)
	}

	return &apiServerDB, nil
}

var _ Database = &database{}

func (db *database) ReadDevices(ctx context.Context) ([]*pb.Device, error) {
	rows, err := db.queries.GetDevices(ctx)
	if err != nil {
		return nil, err
	}

	devices := make([]*pb.Device, 0)
	for _, row := range rows {
		issues, err := db.getDeviceIssues(ctx, row)
		if err != nil {
			continue
		}

		device, err := db.sqlcDeviceToPbDevice(row, issues)
		if err != nil {
			return nil, fmt.Errorf("converting device %v: %w", row.ID, err)
		}
		devices = append(devices, device)
	}

	return devices, nil
}

func (db *database) ReadPeers(ctx context.Context) ([]*peer, error) {
	rows, err := db.queries.GetPeers(ctx)
	if err != nil {
		return nil, err
	}

	peers := make([]*peer, len(rows))
	for i, row := range rows {
		peers[i] = &peer{
			name:      row.Username,
			publicKey: row.PublicKey,
			ip:        row.Ipv4,
		}
	}

	return peers, nil
}

func (db *database) SetDeviceSeenByKolide(ctx context.Context, externalID string, serial string, platform string, lastSeen *time.Time) error {
	lastSeenCol := sql.NullString{}
	if lastSeen != nil {
		lastSeenCol = sql.NullString{
			String: timeToString(*lastSeen),
			Valid:  true,
		}
	}

	return db.queries.UpdateDevice(ctx, sqlc.UpdateDeviceParams{
		Healthy:  false,
		Serial:   serial,
		Platform: platform,
		LastUpdated: sql.NullString{
			String: timeToString(time.Now().UTC()),
			Valid:  true,
		},
		LastSeen: lastSeenCol,
		ExternalID: sql.NullString{
			String: externalID,
			Valid:  externalID != "",
		},
	})
}

func (db *database) UpdateDevices(ctx context.Context, devices []*pb.Device) error {
	var errs error
	for _, device := range devices {
		lastSeen := sql.NullString{}
		if device.LastSeen != nil {
			lastSeen = sql.NullString{
				String: timeToString(device.LastSeen.AsTime().UTC()),
				Valid:  true,
			}
		}

		errs = errors.Join(db.queries.UpdateDevice(ctx, sqlc.UpdateDeviceParams{
			Serial:   device.Serial,
			Platform: device.Platform,
			LastUpdated: sql.NullString{
				String: timeToString(time.Now().UTC()),
				Valid:  true,
			},
			LastSeen: lastSeen,
			ExternalID: sql.NullString{
				String: device.ExternalID,
				Valid:  device.ExternalID != "",
			},
		}))

	}

	return errs
}

func (db *database) UpdateGateway(ctx context.Context, gw *pb.Gateway) error {
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

func (db *database) UpdateGatewayDynamicFields(ctx context.Context, gw *pb.Gateway) error {
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

func (db *database) AddGateway(ctx context.Context, gw *pb.Gateway) error {
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

func (db *database) AddDevice(ctx context.Context, device *pb.Device) error {
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

	initialHealthy := !db.kolideEnabled

	err = db.queries.AddDevice(ctx, sqlc.AddDeviceParams{
		Serial:    device.Serial,
		Username:  device.Username,
		PublicKey: device.PublicKey,
		Ipv4:      availableIpV4,
		Ipv6:      availableIpV6,
		Healthy:   initialHealthy,
		Platform:  device.Platform,
	})
	if err != nil {
		return fmt.Errorf("inserting new device: %w", err)
	}

	return nil
}

func (db *database) ReadDevice(ctx context.Context, publicKey string) (*pb.Device, error) {
	row, err := db.queries.GetDeviceByPublicKey(ctx, publicKey)
	if err != nil {
		return nil, err
	}

	issues, err := db.getDeviceIssues(ctx, row)
	if err != nil {
		return nil, err
	}

	return db.sqlcDeviceToPbDevice(row, issues)
}

func (db *database) ReadDeviceById(ctx context.Context, deviceID int64) (*pb.Device, error) {
	row, err := db.queries.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	issues, err := db.getDeviceIssues(ctx, row)
	if err != nil {
		return nil, err
	}

	return db.sqlcDeviceToPbDevice(row, issues)
}

func (db *database) ReadDeviceByExternalID(ctx context.Context, externalID string) (*pb.Device, error) {
	id := sql.NullString{
		String: externalID,
		Valid:  externalID != "",
	}
	row, err := db.queries.GetDeviceByExternalID(ctx, id)
	if err != nil {
		return nil, err
	}

	issues, err := db.getDeviceIssues(ctx, row)
	if err != nil {
		return nil, err
	}

	return db.sqlcDeviceToPbDevice(row, issues)
}

func (db *database) ReadGateways(ctx context.Context) ([]*pb.Gateway, error) {
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

func (db *database) ReadGateway(ctx context.Context, name string) (*pb.Gateway, error) {
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

func (db *database) readExistingIPs(ctx context.Context) ([]string, error) {
	var ips []string

	devices, err := db.ReadPeers(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading devices: %w", err)
	}

	for _, device := range devices {
		ips = append(ips, device.ip)
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

func (db *database) ReadDeviceBySerialPlatform(ctx context.Context, serial, platform string) (*pb.Device, error) {
	row, err := db.queries.GetDeviceBySerialAndPlatform(ctx, sqlc.GetDeviceBySerialAndPlatformParams{
		Serial:   serial,
		Platform: platform,
	})
	if err != nil {
		return nil, err
	}

	issues, err := db.getDeviceIssues(ctx, row)
	if err != nil {
		return nil, err
	}

	return db.sqlcDeviceToPbDevice(row, issues)
}

func (db *database) AddSessionInfo(ctx context.Context, si *pb.Session) error {
	err := db.queries.Transaction(ctx, func(ctx context.Context, qtx *sqlc.Queries) error {
		err := qtx.AddSession(ctx, sqlc.AddSessionParams{
			Key:      si.Key,
			Expiry:   timeToString(si.Expiry.AsTime().UTC()),
			DeviceID: si.GetDevice().GetId(),
			ObjectID: si.ObjectID,
		})
		if err != nil {
			db.log.WithError(err).WithField("device", si.GetDevice()).WithField("session", si).Error("storing session")
			return fmt.Errorf("storing session: %w", err)
		}

		for _, groupID := range si.Groups {
			err = qtx.AddSessionAccessGroupID(ctx, sqlc.AddSessionAccessGroupIDParams{
				SessionKey: si.Key,
				GroupID:    groupID,
			})
			if err != nil {
				return fmt.Errorf("storing session group: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

func (db *database) ReadSessionInfo(ctx context.Context, key string) (*pb.Session, error) {
	row, err := db.queries.GetSessionByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	groupIDs, err := db.queries.GetSessionGroupIDs(ctx, key)
	if err != nil {
		return nil, err
	}

	issues, err := db.getDeviceIssues(ctx, &row.Device)
	if err != nil {
		return nil, fmt.Errorf("getting device issues: %w", err)
	}

	device, err := db.sqlcDeviceToPbDevice(&row.Device, issues)
	if err != nil {
		return nil, fmt.Errorf("converting device: %w", err)
	}

	return db.sqlcSessionAndDeviceToPbSession(row.Session, device, groupIDs), nil
}

func (db *database) ReadSessionInfos(ctx context.Context) ([]*pb.Session, error) {
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

		issues, err := db.getDeviceIssues(ctx, &row.Device)
		if err != nil {
			return nil, fmt.Errorf("getting device issues: %w", err)
		}

		device, err := db.sqlcDeviceToPbDevice(&row.Device, issues)
		if err != nil {
			return nil, fmt.Errorf("converting device: %w", err)
		}

		sessions = append(sessions, db.sqlcSessionAndDeviceToPbSession(row.Session, device, groupIDs))
	}

	return sessions, nil
}

func (db *database) ReadMostRecentSessionInfo(ctx context.Context, deviceID int64) (*pb.Session, error) {
	row, err := db.queries.GetMostRecentDeviceSession(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	groupIDs, err := db.queries.GetSessionGroupIDs(ctx, row.Session.Key)
	if err != nil {
		return nil, err
	}

	issues, err := db.getDeviceIssues(ctx, &row.Device)
	if err != nil {
		return nil, fmt.Errorf("getting device issues: %w", err)
	}

	device, err := db.sqlcDeviceToPbDevice(&row.Device, issues)
	if err != nil {
		return nil, fmt.Errorf("converting device: %w", err)
	}

	return db.sqlcSessionAndDeviceToPbSession(row.Session, device, groupIDs), nil
}

func (db *database) getNextAvailableIPv4(ctx context.Context) (string, error) {
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

func (db *database) getNextAvailableIPv6(ctx context.Context) (string, error) {
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

func (db *database) RemoveExpiredSessions(ctx context.Context) error {
	return db.queries.RemoveExpiredSessions(ctx)
}

func (db *database) sqlcDeviceToPbDevice(sqlcDevice *sqlc.Device, issues []*pb.DeviceIssue) (*pb.Device, error) {
	pbDevice := &pb.Device{
		Id:         int64(sqlcDevice.ID),
		Serial:     sqlcDevice.Serial,
		PublicKey:  sqlcDevice.PublicKey,
		Ipv4:       sqlcDevice.Ipv4,
		Ipv6:       sqlcDevice.Ipv6,
		Username:   sqlcDevice.Username,
		ExternalID: sqlcDevice.ExternalID.String,
		Platform:   string(sqlcDevice.Platform),
		Issues:     issues,
	}

	if sqlcDevice.LastUpdated.Valid {
		pbDevice.LastUpdated = timestamppb.New(stringToTime(sqlcDevice.LastUpdated.String))
	}
	if sqlcDevice.LastSeen.Valid {
		pbDevice.LastSeen = timestamppb.New(stringToTime(sqlcDevice.LastSeen.String))
	}

	return pbDevice, nil
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

func (db *database) sqlcSessionAndDeviceToPbSession(s sqlc.Session, device *pb.Device, groupIDs []string) *pb.Session {
	return &pb.Session{
		Key:      s.Key,
		Device:   device,
		ObjectID: s.ObjectID,
		Expiry:   timestamppb.New(stringToTime(s.Expiry)),
		Groups:   groupIDs,
	}
}
