package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/pb"
)

type apiServerDB struct {
	conn                *sql.DB
	IPAllocator         IPAllocator
	defaultDeviceHealth bool
}

var _ APIServer = &apiServerDB{}

func New(dsn, driver string, ipAllocator IPAllocator, defaultDeviceHealth bool) (*apiServerDB, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %s", err)
	}

	apiServerDB := apiServerDB{
		conn:                db,
		IPAllocator:         ipAllocator,
		defaultDeviceHealth: defaultDeviceHealth,
	}

	ctx := context.Background()
	for backoff := 0; backoff < 5; backoff++ {
		time.Sleep(time.Duration(backoff) * time.Second)
		err = apiServerDB.Migrate(ctx)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("migrating database: %w", err)
	}

	return &apiServerDB, nil
}

func (db *apiServerDB) ReadDevices(ctx context.Context) ([]*pb.Device, error) {
	query := fmt.Sprintf("SELECT %s FROM device;", DeviceFields)

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying for devices: %v", err)
	}
	defer rows.Close()

	if rows.Err() != nil {
		return nil, fmt.Errorf("querying for devices: %v", rows.Err())
	}

	devices := make([]*pb.Device, 0) // don't want nil declaration here as this JSON encodes to 'null' instead of '[]'
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func (db *apiServerDB) UpdateDevices(ctx context.Context, devices []*pb.Device) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start transaction: %s", err)
	}

	defer tx.Rollback()

	query := `
UPDATE device
   SET healthy = $1, kolide_last_seen = $2, last_updated = NOW()
 WHERE serial = $3 AND platform = $4;`

	for _, device := range devices {
		_, err = tx.ExecContext(ctx, query, device.Healthy, device.KolideLastSeen.AsTime(), device.Serial, device.Platform)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commiting transaction: %w", err)
	}

	log.Debugf("Successfully updated device statuses")

	return nil
}

var mux sync.Mutex

func (db *apiServerDB) UpdateGateway(ctx context.Context, gw *pb.Gateway) error {
	statement := `
UPDATE gateway
    SET public_key = $1,
        access_group_ids = $2,
        endpoint = $3,
        ip = $4,
        routes = $5,
        requires_privileged_access = $6,
        password_hash = $7
 WHERE name = $8;`

	_, err := db.conn.ExecContext(ctx, statement,
		gw.PublicKey,
		strings.Join(gw.AccessGroupIDs, ","),
		gw.Endpoint,
		gw.Ip,
		strings.Join(gw.Routes, ","),
		gw.RequiresPrivilegedAccess,
		gw.PasswordHash,
		gw.Name)
	if err != nil {
		return fmt.Errorf("updating gateway: %w", err)
	}

	log.Debugf("Updated gateway: %s", gw.Name)
	return nil
}

func (db *apiServerDB) UpdateGatewayDynamicFields(ctx context.Context, gw *pb.Gateway) error {
	statement := `
UPDATE gateway
    SET access_group_ids = $1,
        routes = $2,
        requires_privileged_access = $3
 WHERE name = $4;`

	_, err := db.conn.ExecContext(ctx, statement,
		strings.Join(gw.AccessGroupIDs, ","),
		strings.Join(gw.Routes, ","),
		gw.RequiresPrivilegedAccess,
		gw.Name)
	if err != nil {
		return fmt.Errorf("updating gateway dynamic fields: %w", err)
	}

	log.Debugf("Updated gateway dynamic fields: %s", gw.Name)
	return nil
}

func (db *apiServerDB) AddGateway(ctx context.Context, gw *pb.Gateway) error {
	mux.Lock()
	defer mux.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("start transaction: %w", err)
	}

	defer tx.Rollback()
	takenIps, err := db.readExistingIPs(ctx)
	if err != nil {
		return fmt.Errorf("reading existing ips: %w", err)
	}

	availableIp, err := db.IPAllocator.NextIP(takenIps)
	if err != nil {
		return fmt.Errorf("finding available ip: %w", err)
	}

	statement := `
INSERT INTO gateway (name, endpoint, public_key, ip, password_hash, access_group_ids, routes, requires_privileged_access)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (name) DO UPDATE SET endpoint = EXCLUDED.endpoint, public_key = EXCLUDED.public_key, password_hash = EXCLUDED.password_hash;`

	_, err = db.conn.ExecContext(ctx, statement,
		gw.Name,
		gw.Endpoint,
		gw.PublicKey,
		availableIp,
		gw.PasswordHash,
		strings.Join(gw.AccessGroupIDs, ","),
		strings.Join(gw.Routes, ","),
		gw.RequiresPrivilegedAccess,
	)
	if err != nil {
		return fmt.Errorf("inserting new gateway, statement: '%s', error: %w", statement, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	log.Infof("Added gateway: %+v", gw.Name)
	return nil
}

func (db *apiServerDB) AddDevice(ctx context.Context, device *pb.Device) error {
	mux.Lock()
	defer mux.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("start transaction: %w", err)
	}

	defer tx.Rollback()
	ips, err := db.readExistingIPs(ctx)
	if err != nil {
		return fmt.Errorf("reading existing ips: %w", err)
	}

	ip, err := db.IPAllocator.NextIP(ips)
	if err != nil {
		return fmt.Errorf("finding available ip: %w", err)
	}

	statement := `
INSERT INTO device (serial, username, public_key, ip, healthy, psk, platform)
            VALUES ($1, $2, $3, $4, $5, '', $6)
ON CONFLICT(serial, platform) DO UPDATE SET username = $2, public_key = $3;`
	_, err = tx.ExecContext(ctx, statement, device.Serial, device.Username, device.PublicKey, ip, db.defaultDeviceHealth, device.Platform)
	if err != nil {
		return fmt.Errorf("inserting new device: %w", err)
	}

	stmt := `
SELECT id
  FROM device
 WHERE serial = $1
   AND platform = $2;`

	row := tx.QueryRowContext(ctx, stmt, device.Serial, device.Platform)
	if row.Err() != nil {
		return fmt.Errorf("querying for id: %w", err)
	}

	err = row.Scan(&device.Id)
	if err != nil {
		return fmt.Errorf("scanning row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commiting transaction: %w", err)
	}

	log.Infof("Added or updated device: %+v", device)
	return nil
}

func (db *apiServerDB) ReadDevice(ctx context.Context, publicKey string) (*pb.Device, error) {
	query := fmt.Sprintf("SELECT %s FROM device WHERE public_key = $1;", DeviceFields)

	row := db.conn.QueryRowContext(ctx, query, publicKey)

	return scanDevice(row)
}

func (db *apiServerDB) ReadDeviceById(ctx context.Context, deviceID int64) (*pb.Device, error) {
	query := fmt.Sprintf("SELECT %s FROM device WHERE id = $1;", DeviceFields)

	row := db.conn.QueryRowContext(ctx, query, deviceID)

	return scanDevice(row)
}

func (db *apiServerDB) ReadGateways(ctx context.Context) ([]*pb.Gateway, error) {
	query := fmt.Sprintf("SELECT %s FROM gateway;", GatewayFields)

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying for gateways %w", err)
	}
	defer rows.Close()

	var gateways []*pb.Gateway
	for rows.Next() {
		gateway, err := scanGateway(rows)
		if err != nil {
			return nil, fmt.Errorf("scan gateway: %w", err)
		}

		gateways = append(gateways, gateway)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterating over rows: %w", rows.Err())
	}

	return gateways, nil
}

func (db *apiServerDB) ReadGateway(ctx context.Context, name string) (*pb.Gateway, error) {
	query := fmt.Sprintf("SELECT %s FROM gateway WHERE name = $1;", GatewayFields)
	row := db.conn.QueryRowContext(ctx, query, name)

	return scanGateway(row)
}

func (db *apiServerDB) readExistingIPs(ctx context.Context) ([]string, error) {
	var ips []string

	if devices, err := db.ReadDevices(ctx); err != nil {
		return nil, fmt.Errorf("reading devices: %w", err)
	} else {
		for _, device := range devices {
			ips = append(ips, device.Ip)
		}
	}

	if gateways, err := db.ReadGateways(ctx); err != nil {
		return nil, fmt.Errorf("reading gateways: %w", err)
	} else {
		for _, gateway := range gateways {
			ips = append(ips, gateway.Ip)
		}
	}

	return ips, nil
}

func (db *apiServerDB) ReadDeviceBySerialPlatform(ctx context.Context, serial string, platform string) (*pb.Device, error) {
	query := fmt.Sprintf(`
SELECT %s
  FROM device
 WHERE serial = $1
   AND platform = $2`, DeviceFields)

	row := db.conn.QueryRowContext(ctx, query, serial, platform)

	return scanDevice(row)
}

func (db *apiServerDB) AddSessionInfo(ctx context.Context, si *pb.Session) error {
	query := `
INSERT INTO session (key, expiry, device_id, groups, object_id)
             VALUES ($1, $2, $3, $4, $5);`

	_, err := db.conn.ExecContext(ctx, query, si.Key, si.Expiry.AsTime(), si.GetDevice().GetId(), strings.Join(si.Groups, ","), si.ObjectID)
	if err != nil {
		return fmt.Errorf("scanning row: %s", err)
	}

	log.Debugf("persisted session: %v", si)

	return nil
}

func (db *apiServerDB) ReadSessionInfo(ctx context.Context, key string) (*pb.Session, error) {
	query := fmt.Sprintf("SELECT %s FROM session WHERE key = $1;", sessionFields)

	row := db.conn.QueryRowContext(ctx, query, key)

	return scanSessionWithDevice(ctx, row, db)
}

func (db *apiServerDB) ReadSessionInfos(ctx context.Context) ([]*pb.Session, error) {
	query := fmt.Sprintf("SELECT %s FROM session WHERE expiry > now();", sessionFields)

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query rows: %w", err)
	}
	defer rows.Close()

	sessionInfos := make([]*pb.Session, 0)
	for rows.Next() {
		if rows.Err() != nil {
			return nil, fmt.Errorf("read row: %w", rows.Err())
		}

		session, err := scanSessionWithDevice(ctx, rows, db)
		if err != nil {
			return nil, err
		}

		sessionInfos = append(sessionInfos, session)
	}

	log.Debugf("retrieved session infos from db")

	return sessionInfos, nil
}

func (db *apiServerDB) ReadMostRecentSessionInfo(ctx context.Context, deviceID int64) (*pb.Session, error) {
	query := fmt.Sprintf(`
SELECT %s
 FROM session
WHERE device_id = $1
ORDER BY expiry DESC
LIMIT 1;`, sessionFields)

	row := db.conn.QueryRowContext(ctx, query, deviceID)

	return scanSessionWithDevice(ctx, row, db)
}

func (db *apiServerDB) Migrate(ctx context.Context) error {
	var currentVersion int

	query := "SELECT MAX(version) FROM migrations"
	row := db.conn.QueryRowContext(ctx, query)
	err := row.Scan(&currentVersion)
	if err != nil {
		// error might be due to no schema.
		// no way to detect this, so log error and continue with migrations.
		log.Warnf("unable to get current migration version: %s", err)
	}

	migrations, err := migrations()
	if err != nil {
		return fmt.Errorf("unable to read migrations: %w", err)
	}
	for _, migration := range migrations {
		if migration.version <= currentVersion {
			continue
		}

		log.Infof("migrating database schema to version %d", migration.version)

		_, err = db.conn.ExecContext(ctx, migration.sql)
		if err != nil {
			return fmt.Errorf("migrating to version %d: %s", migration.version, err)
		}
	}

	return nil
}
