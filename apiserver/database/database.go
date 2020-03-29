package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

type APIServerDB interface {
	ReadClients() ([]Client, error)
}

type Client struct {
	PSK string `json:"psk"`
	Peer
}

type Peer struct {
	PublicKey string `json:"public_key"`
	IP        string `json:"ip"`
}

type database struct {
	conn *pgxpool.Pool
}

func New(dsn string) (APIServerDB, error) {
	ctx := context.Background()
	conn, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %s", err)
	}

	return &database{conn: conn}, nil
}

func (d *database) ReadClients() (clients []Client, err error) {
	ctx := context.Background()

	query := `
            SELECT public_key, ip, psk from peer
              JOIN client c on peer.id = c.peer_id
              JOIN ip i on peer.id = i.peer_id
	`

	rows, err := d.conn.Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("querying for clients: %s", err)
	}

	for rows.Next() {
		var client Client

		err := rows.Scan(&client.PublicKey, &client.IP, &client.PSK)

		if err != nil {
			return nil, fmt.Errorf("scanning row: %s", err)
		}

		clients = append(clients, client)
	}

	return
}
