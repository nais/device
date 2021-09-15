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

	"github.com/nais/device/pkg/apiserver/cidr"
	"github.com/nais/device/pkg/pb"
)

const (
	TunnelCidr = "10.255.240.0/21"
)

type apiServerDB struct {
	conn *sql.DB
}

var _ APIServer = &apiServerDB{}

func New(dsn, driver string) (*apiServerDB, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %s", err)
	}

	apiServerDB := apiServerDB{conn: db}

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

func (db *apiServerDB) ReadDevices() ([]*pb.Device, error) {
	ctx := context.Background()

	query := `
SELECT id, serial, username, psk, platform, last_updated, kolide_last_seen, healthy, public_key, ip
FROM device;`

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
           SET healthy = $1, kolide_last_seen = $2, last_updated = CAST(EXTRACT(EPOCH FROM NOW()) AS BIGINT)
         WHERE serial = $3 AND platform = $4;
    `

	for _, device := range devices {
		_, err = tx.ExecContext(ctx, query, device.Healthy, device.KolideLastSeen.AsTime().Unix(), device.Serial, device.Platform)
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

func (db *apiServerDB) UpdateGateway(ctx context.Context, name string, routes, accessGroupIDs []string, requiresPrivilegedAccess bool) error {
	statement := `
UPDATE gateway 
SET routes = $1, access_group_ids = $2, requires_privileged_access = $3
WHERE name = $4;`

	_, err := db.conn.ExecContext(ctx, statement, strings.Join(routes, ","), strings.Join(accessGroupIDs, ","), requiresPrivilegedAccess, name)
	if err != nil {
		return fmt.Errorf("updating gateway: %w", err)
	}

	log.Debugf("Updated gateway: %s", name)
	return nil
}

func (db *apiServerDB) AddGateway(ctx context.Context, name, endpoint, publicKey string) error {
	mux.Lock()
	defer mux.Unlock()

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("start transaction: %w", err)
	}

	defer tx.Rollback()
	takenIps, err := db.readExistingIPs()
	if err != nil {
		return fmt.Errorf("reading existing ips: %w", err)
	}

	availableIp, err := cidr.FindAvailableIP(TunnelCidr, takenIps)
	if err != nil {
		return fmt.Errorf("finding available ip: %w", err)
	}

	statement := `
INSERT INTO gateway (name, endpoint, public_key, ip)
VALUES ($1, $2, $3, $4);`

	_, err = db.conn.ExecContext(ctx, statement, name, endpoint, publicKey, availableIp)
	if err != nil {
		return fmt.Errorf("inserting new gateway, statement: '%s', error: %w", statement, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	log.Infof("Added gateway: %+v", name)
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
	ips, err := db.readExistingIPs()
	if err != nil {
		return fmt.Errorf("reading existing ips: %w", err)
	}

	ip, err := cidr.FindAvailableIP(TunnelCidr, ips)
	if err != nil {
		return fmt.Errorf("finding available ip: %w", err)
	}

	statement := `
INSERT INTO device (serial, username, public_key, ip, healthy, psk, platform)
VALUES ($1, $2, $3, $4, false, '', $5)
ON CONFLICT(serial, platform) DO UPDATE SET username = $2, public_key = $3;`
	_, err = tx.ExecContext(ctx, statement, device.Serial, device.Username, device.PublicKey, ip, device.Platform)
	if err != nil {
		return fmt.Errorf("inserting new device: %w", err)
	}

	stmt := `SELECT id FROM device WHERE serial = $1 AND platform = $2`
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

func (db *apiServerDB) ReadDevice(publicKey string) (*pb.Device, error) {
	ctx := context.Background()

	query := `
SELECT id, serial, username, psk, platform, last_updated, kolide_last_seen, healthy, public_key, ip
  FROM device
 WHERE public_key = $1;`

	row := db.conn.QueryRowContext(ctx, query, publicKey)

	return scanDevice(row)
}

func (db *apiServerDB) ReadDeviceById(ctx context.Context, deviceID int64) (*pb.Device, error) {
	query := `
SELECT id, serial, username, psk, platform, last_updated, kolide_last_seen, healthy, public_key, ip
  FROM device
 WHERE id = $1;`

	row := db.conn.QueryRowContext(ctx, query, deviceID)

	return scanDevice(row)
}

func (db *apiServerDB) ReadGateways() ([]*pb.Gateway, error) {
	ctx := context.Background()

	query := `
SELECT public_key, access_group_ids, endpoint, ip, routes, name, requires_privileged_access
  FROM gateway;`

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying for gateways %w", err)
	}

	var gateways []*pb.Gateway
	for rows.Next() {
		gateway := &pb.Gateway{}
		var routes string
		var accessGroupIDs string
		err := rows.Scan(&gateway.PublicKey, &accessGroupIDs, &gateway.Endpoint, &gateway.Ip, &routes, &gateway.Name, &gateway.RequiresPrivilegedAccess)
		if err != nil {
			return nil, fmt.Errorf("scanning gateway: %w", err)
		}

		if len(accessGroupIDs) != 0 {
			gateway.AccessGroupIDs = strings.Split(accessGroupIDs, ",")
		}

		if len(routes) != 0 {
			gateway.Routes = strings.Split(routes, ",")
		}

		gateways = append(gateways, gateway)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("iterating over rows: %w", rows.Err())
	}

	return gateways, nil

}

func (db *apiServerDB) ReadGateway(name string) (*pb.Gateway, error) {
	ctx := context.Background()

	query := `
SELECT public_key, access_group_ids, endpoint, ip, routes, name, requires_privileged_access
  FROM gateway
 WHERE name = $1;`

	row := db.conn.QueryRowContext(ctx, query, name)

	var gateway pb.Gateway
	var routes string
	var accessGroupIDs string
	err := row.Scan(&gateway.PublicKey, &accessGroupIDs, &gateway.Endpoint, &gateway.Ip, &routes, &gateway.Name, &gateway.RequiresPrivilegedAccess)
	if err != nil {
		return nil, fmt.Errorf("scanning gateway: %w", err)
	}

	if len(accessGroupIDs) != 0 {
		gateway.AccessGroupIDs = strings.Split(accessGroupIDs, ",")
	}

	if len(routes) != 0 {
		gateway.Routes = strings.Split(routes, ",")
	}

	return &gateway, nil
}

func (db *apiServerDB) readExistingIPs() ([]string, error) {
	ips := []string{
		"10.255.240.1", // reserve apiserver ip
	}

	if devices, err := db.ReadDevices(); err != nil {
		return nil, fmt.Errorf("reading devices: %w", err)
	} else {
		for _, device := range devices {
			ips = append(ips, device.Ip)
		}
	}

	if gateways, err := db.ReadGateways(); err != nil {
		return nil, fmt.Errorf("reading gateways: %w", err)
	} else {
		for _, gateway := range gateways {
			ips = append(ips, gateway.Ip)
		}
	}

	return ips, nil
}

func (db *apiServerDB) ReadDeviceBySerialPlatformUsername(ctx context.Context, serial string, platform string, username string) (*pb.Device, error) {
	query := `
SELECT id, username, serial, psk, platform, healthy, last_updated, kolide_last_seen, public_key, ip
  FROM device
 WHERE serial = $1
   AND platform = $2
   AND lower(username) = $3;
	`

	lowerUsername := strings.ToLower(username)
	row := db.conn.QueryRowContext(ctx, query, serial, platform, lowerUsername)

	return scanDevice(row)
}

func (db *apiServerDB) AddSessionInfo(ctx context.Context, si *pb.Session) error {
	query := `
INSERT INTO session (key, expiry, device_id, groups, object_id)
             VALUES ($1, $2, $3, $4, $5);
`

	_, err := db.conn.ExecContext(ctx, query, si.Key, si.Expiry.AsTime().Unix(), si.GetDevice().GetId(), strings.Join(si.Groups, ","), si.ObjectID)
	if err != nil {
		return fmt.Errorf("scanning row: %s", err)
	}

	log.Debugf("persisted session: %v", si)

	return nil
}

func (db *apiServerDB) ReadSessionInfo(ctx context.Context, key string) (*pb.Session, error) {
	query := `
SELECT key, expiry, device_id, groups, object_id
FROM session
WHERE key = $1;
`

	row := db.conn.QueryRowContext(ctx, query, key)

	return scanSessionWithDevice(ctx, row, db)
}

func (db *apiServerDB) ReadSessionInfos(ctx context.Context) ([]*pb.Session, error) {
	query := `
SELECT key, expiry, device_id, groups, object_id
FROM session
WHERE to_timestamp(expiry) > now();
`

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query rows: %w", err)
	}

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
	query := `
SELECT key, expiry, device_id, groups, object_id
FROM session
WHERE device_id = $1
ORDER BY expiry DESC
LIMIT 1;
`

	row := db.conn.QueryRowContext(ctx, query, deviceID)

	return scanSessionWithDevice(ctx, row, db)
}

func (db *apiServerDB) Migrate(ctx context.Context) error {
	var version int

	query := `SELECT MAX(version) FROM migrations`
	row := db.conn.QueryRowContext(ctx, query)
	err := row.Scan(&version)

	if err != nil {
		// error might be due to no schema.
		// no way to detect this, so log error and continue with migrations.
		log.Warnf("unable to get current migration version: %s", err)
	}

	for version < len(migrations) {
		log.Infof("migrating database schema to version %d", version+1)

		_, err = db.conn.ExecContext(ctx, migrations[version])
		if err != nil {
			return fmt.Errorf("migrating to version %d: %s", version+1, err)
		}

		version++
	}

	return nil
}
