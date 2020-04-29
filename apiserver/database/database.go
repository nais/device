package database

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/nais/device/apiserver/cidr"
	"github.com/nais/device/pkg/random"
	log "github.com/sirupsen/logrus"
)

const (
	TunnelCidr = "10.255.240.0/21"
)

type APIServerDB struct {
	conn *pgxpool.Pool
}

type Device struct {
	Serial    string     `json:"serial"`
	PSK       string     `json:"psk"`
	LastCheck *time.Time `json:"lastCheck"`
	Healthy   *bool      `json:"isHealthy"`
	PublicKey string     `json:"publicKey"`
	IP        string     `json:"ip"`
	Username  string     `json:"username"`
}

type Gateway struct {
	Endpoint  string   `json:"endpoint"`
	PublicKey string   `json:"publicKey"`
	IP        string   `json:"ip"`
	Routes    []string `json:"routes"`
}

// NewTestDatabase creates and returns a new nais device database within the provided database instance
func NewTestDatabase(dsn, schema string) (*APIServerDB, error) {
	ctx := context.Background()
	initialConn, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %v", err)
	}

	defer initialConn.Close()

	databaseName := random.RandomString(5, random.LowerCaseLetters)

	_, err = initialConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %v", databaseName))
	if err != nil {
		return nil, fmt.Errorf("creating database: %v", err)
	}

	conn, err := pgxpool.Connect(ctx, fmt.Sprintf("%s/%s", dsn, databaseName))
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %v", err)
	}

	b, err := ioutil.ReadFile(schema)
	if err != nil {
		return nil, fmt.Errorf("reading schema file from disk: %w", err)
	}

	_, err = conn.Exec(ctx, string(b))
	if err != nil {
		return nil, fmt.Errorf("executing schema: %v", err)
	}

	return &APIServerDB{conn: conn}, nil
}

func New(dsn string) (*APIServerDB, error) {
	ctx := context.Background()
	conn, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %s", err)
	}

	return &APIServerDB{conn: conn}, nil
}

func (d *APIServerDB) ReadDevices() (devices []Device, err error) {
	ctx := context.Background()

	query := `
SELECT public_key, username, ip, psk, serial, healthy, last_check
FROM device;`

	rows, err := d.conn.Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("querying for devices: %v", err)
	}

	defer rows.Close()

	if rows.Err() != nil {
		return nil, fmt.Errorf("querying for devices: %v", rows.Err())
	}

	for rows.Next() {
		var device Device

		err := rows.Scan(&device.PublicKey, &device.Username, &device.IP, &device.PSK, &device.Serial, &device.Healthy, &device.LastCheck)

		if err != nil {
			return nil, fmt.Errorf("scanning row: %s", err)
		}

		devices = append(devices, device)
	}

	return
}

func (d *APIServerDB) UpdateDeviceStatus(devices []Device) error {
	ctx := context.Background()

	tx, err := d.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("start transaction: %s", err)
	}

	defer tx.Rollback(ctx)

	query := `
		UPDATE device
           SET healthy = $1, last_check = NOW()
         WHERE serial = $2;
    `

	for _, device := range devices {
		_, err = tx.Exec(ctx, query, device.Healthy, device.Serial)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commiting transaction: %w", err)
	}

	log.Infof("Successfully updated device statuses")

	return nil
}

var mux sync.Mutex

func (d *APIServerDB) AddDevice(username, publicKey, serial string) error {
	mux.Lock()
	defer mux.Unlock()

	ctx := context.Background()

	tx, err := d.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("start transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	ips, err := d.readExistingIPs()
	if err != nil {
		return fmt.Errorf("reading existing ips: %w", err)
	}

	ip, err := cidr.FindAvailableIP(TunnelCidr, ips)
	if err != nil {
		return fmt.Errorf("finding available ip: %w", err)
	}

	statement := `
INSERT INTO device (serial, username, public_key, ip, healthy, psk)
VALUES ($1, $2, $3, $4, false, '')
ON CONFLICT(serial) DO UPDATE SET username = $2, public_key = $3;`
	_, err = tx.Exec(ctx, statement, serial, username, publicKey, ip)

	if err != nil {
		return fmt.Errorf("inserting new device: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commiting transaction: %w", err)
	}

	log.Infof("Added or updated device with serial %v for user %v with public key %v to database.", serial, username, publicKey)

	return nil
}

func (d *APIServerDB) ReadDevice(publicKey string) (*Device, error) {
	ctx := context.Background()

	query := `
SELECT serial, username, psk, last_check, healthy, public_key, ip
  FROM device
 WHERE public_key = $1;`

	row := d.conn.QueryRow(ctx, query, publicKey)

	var device Device
	err := row.Scan(&device.Serial, &device.Username, &device.PSK, &device.LastCheck, &device.Healthy, &device.PublicKey, &device.IP)

	if err != nil {
		return nil, fmt.Errorf("scanning row: %s", err)
	}

	return &device, nil
}

func (d *APIServerDB) ReadGateways() ([]Gateway, error) {
	ctx := context.Background()

	query := `
SELECT public_key, endpoint, ip, routes
  FROM gateway;`

	rows, err := d.conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying for gateways %w", err)
	}

	var gateways []Gateway
	for rows.Next() {
		var gateway Gateway
		var routes string
		err := rows.Scan(&gateway.PublicKey, &gateway.Endpoint, &gateway.IP, &routes)
		if err != nil {
			return nil, fmt.Errorf("scanning gateway: %w", err)
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

func (d *APIServerDB) ReadGateway(publicKey string) (*Gateway, error) {
	ctx := context.Background()

	query := `
SELECT public_key, endpoint, ip, routes
  FROM gateway
 WHERE public_key = $1;`

	row := d.conn.QueryRow(ctx, query, publicKey)

	var gateway Gateway
	var routes string
	err := row.Scan(&gateway.PublicKey, &gateway.Endpoint, &gateway.IP, &routes)
	if err != nil {
		return nil, fmt.Errorf("scanning gateway: %w", err)
	}

	if len(routes) != 0 {
		gateway.Routes = strings.Split(routes, ",")
	}

	return &gateway, nil
}

func (d *APIServerDB) readExistingIPs() ([]string, error) {
	ips := []string{
		"10.255.240.1", // reserve api server ip
	}

	if devices, err := d.ReadDevices(); err != nil {
		return nil, fmt.Errorf("reading devices: %w", err)
	} else {
		for _, device := range devices {
			ips = append(ips, device.IP)
		}
	}

	if gateways, err := d.ReadGateways(); err != nil {
		return nil, fmt.Errorf("reading gateways: %w", err)
	} else {
		for _, gateway := range gateways {
			ips = append(ips, gateway.IP)
		}
	}

	return ips, nil
}
