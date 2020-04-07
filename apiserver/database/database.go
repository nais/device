package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	ControlPlaneCidr = "10.255.240.0/21"
	//DataPlaneCidr    = "10.255.248.0/21"
)

type APIServerDB struct {
	conn *pgxpool.Pool
}

type Client struct {
	Serial    string     `json:"serial"`
	PSK       string     `json:"psk"`
	LastCheck *time.Time `json:"last_check"`
	Healthy   *bool      `json:"is_healthy"`
	Peer
}

type Peer struct {
	PublicKey string `json:"public_key"`
	IP        string `json:"ip"`
}

func New(dsn string) (*APIServerDB, error) {
	ctx := context.Background()
	conn, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %s", err)
	}

	return &APIServerDB{conn: conn}, nil
}

func (d *APIServerDB) ReadClients() (clients []Client, err error) {
	ctx := context.Background()

	query := `
SELECT public_key, ip, psk, serial, healthy, last_check
FROM client
         JOIN client_peer cp on client.id = cp.client_id
         JOIN peer p on cp.peer_id = p.id
WHERE p.type = 'data';
	`

	rows, err := d.conn.Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("querying for clients: %v", err)
	}

	defer rows.Close()

	if rows.Err() != nil {
		return nil, fmt.Errorf("querying for clients: %v", rows.Err())
	}

	for rows.Next() {
		var client Client

		err := rows.Scan(&client.PublicKey, &client.IP, &client.PSK, &client.Serial, &client.Healthy, &client.LastCheck)

		if err != nil {
			return nil, fmt.Errorf("scanning row: %s", err)
		}

		clients = append(clients, client)
	}

	return
}

func (d *APIServerDB) UpdateClientStatus(clients []Client) error {
	ctx := context.Background()

	tx, err := d.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("start transaction: %s", err)
	}

	defer tx.Rollback(ctx)

	query := `
		UPDATE client
           SET healthy = $1, last_check = NOW()
         WHERE serial = $2;
    `

	for _, client := range clients {
		_, err = tx.Exec(ctx, query, client.Healthy, client.Serial)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

var mux sync.Mutex

func (d *APIServerDB) AddClient(username, publicKey, serial string) error {
	mux.Lock()
	defer mux.Unlock()

	ctx := context.Background()

	tx, err := d.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("start transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	ips, err := ips(tx, ctx)
	if err != nil {
		return fmt.Errorf("fetch ips: %w", err)
	}

	ip, err := FindAvailableIP(ControlPlaneCidr, ips)
	if err != nil {
		return fmt.Errorf("finding available ip: %w", err)
	}

	statement := `
WITH
  client_key AS
    (INSERT INTO client (serial, healthy) VALUES ($1, false) RETURNING id),
  peer_control_key AS
    (INSERT INTO peer (public_key, ip, type) VALUES ($2, $3, 'control') RETURNING id)
INSERT
  INTO client_peer(client_id, peer_id)
  (
    SELECT client_key.id, peer_control_key.id
    FROM client_key, peer_control_key
  );
`
	_, err = tx.Exec(ctx, statement, serial, publicKey, ip)

	if err != nil {
		return fmt.Errorf("inserting new client: %w", err)
	}

	return tx.Commit(ctx)
}

func ips(tx pgx.Tx, ctx context.Context) ([]string, error) {
	rows, err := tx.Query(ctx, "SELECT ip FROM peer;")
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}

	var ips []string
	for rows.Next() {
		var ip string
		err = rows.Scan(&ip)

		if err != nil {
			return nil, fmt.Errorf("scan peers: %w", err)
		}

		ips = append(ips, ip)
	}

	return ips, rows.Err()
}
