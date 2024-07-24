// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: devices.sql

package sqlc

import (
	"context"
	"database/sql"
)

const addDevice = `-- name: AddDevice :exec
INSERT INTO devices (serial, username, public_key, ipv4, ipv6, healthy, platform)
VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)
ON CONFLICT(serial, platform) DO
    UPDATE SET username = excluded.username, public_key = excluded.public_key, ipv6 = excluded.ipv6
`

type AddDeviceParams struct {
	Serial    string
	Username  string
	PublicKey string
	Ipv4      string
	Ipv6      string
	Healthy   bool
	Platform  string
}

func (q *Queries) AddDevice(ctx context.Context, arg AddDeviceParams) error {
	_, err := q.exec(ctx, q.addDeviceStmt, addDevice,
		arg.Serial,
		arg.Username,
		arg.PublicKey,
		arg.Ipv4,
		arg.Ipv6,
		arg.Healthy,
		arg.Platform,
	)
	return err
}

const getDeviceByExternalID = `-- name: GetDeviceByExternalID :one
SELECT id, username, serial, platform, healthy, last_updated, public_key, ipv4, ipv6, last_seen, external_id FROM devices WHERE external_id = ?1
`

func (q *Queries) GetDeviceByExternalID(ctx context.Context, externalID sql.NullString) (*Device, error) {
	row := q.queryRow(ctx, q.getDeviceByExternalIDStmt, getDeviceByExternalID, externalID)
	var i Device
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Serial,
		&i.Platform,
		&i.Healthy,
		&i.LastUpdated,
		&i.PublicKey,
		&i.Ipv4,
		&i.Ipv6,
		&i.LastSeen,
		&i.ExternalID,
	)
	return &i, err
}

const getDeviceByID = `-- name: GetDeviceByID :one
SELECT id, username, serial, platform, healthy, last_updated, public_key, ipv4, ipv6, last_seen, external_id FROM devices WHERE devices.id = ?1
`

func (q *Queries) GetDeviceByID(ctx context.Context, id int64) (*Device, error) {
	row := q.queryRow(ctx, q.getDeviceByIDStmt, getDeviceByID, id)
	var i Device
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Serial,
		&i.Platform,
		&i.Healthy,
		&i.LastUpdated,
		&i.PublicKey,
		&i.Ipv4,
		&i.Ipv6,
		&i.LastSeen,
		&i.ExternalID,
	)
	return &i, err
}

const getDeviceByPublicKey = `-- name: GetDeviceByPublicKey :one
SELECT id, username, serial, platform, healthy, last_updated, public_key, ipv4, ipv6, last_seen, external_id FROM devices WHERE public_key = ?1
`

func (q *Queries) GetDeviceByPublicKey(ctx context.Context, publicKey string) (*Device, error) {
	row := q.queryRow(ctx, q.getDeviceByPublicKeyStmt, getDeviceByPublicKey, publicKey)
	var i Device
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Serial,
		&i.Platform,
		&i.Healthy,
		&i.LastUpdated,
		&i.PublicKey,
		&i.Ipv4,
		&i.Ipv6,
		&i.LastSeen,
		&i.ExternalID,
	)
	return &i, err
}

const getDeviceBySerialAndPlatform = `-- name: GetDeviceBySerialAndPlatform :one
SELECT id, username, serial, platform, healthy, last_updated, public_key, ipv4, ipv6, last_seen, external_id FROM devices WHERE serial = ?1 AND platform = ?2
`

type GetDeviceBySerialAndPlatformParams struct {
	Serial   string
	Platform string
}

func (q *Queries) GetDeviceBySerialAndPlatform(ctx context.Context, arg GetDeviceBySerialAndPlatformParams) (*Device, error) {
	row := q.queryRow(ctx, q.getDeviceBySerialAndPlatformStmt, getDeviceBySerialAndPlatform, arg.Serial, arg.Platform)
	var i Device
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Serial,
		&i.Platform,
		&i.Healthy,
		&i.LastUpdated,
		&i.PublicKey,
		&i.Ipv4,
		&i.Ipv6,
		&i.LastSeen,
		&i.ExternalID,
	)
	return &i, err
}

const getDevices = `-- name: GetDevices :many
SELECT id, username, serial, platform, healthy, last_updated, public_key, ipv4, ipv6, last_seen, external_id FROM devices ORDER BY devices.id
`

func (q *Queries) GetDevices(ctx context.Context) ([]*Device, error) {
	rows, err := q.query(ctx, q.getDevicesStmt, getDevices)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*Device
	for rows.Next() {
		var i Device
		if err := rows.Scan(
			&i.ID,
			&i.Username,
			&i.Serial,
			&i.Platform,
			&i.Healthy,
			&i.LastUpdated,
			&i.PublicKey,
			&i.Ipv4,
			&i.Ipv6,
			&i.LastSeen,
			&i.ExternalID,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPeers = `-- name: GetPeers :many
SELECT username, public_key, ipv4 FROM devices ORDER BY devices.id
`

type GetPeersRow struct {
	Username  string
	PublicKey string
	Ipv4      string
}

func (q *Queries) GetPeers(ctx context.Context) ([]*GetPeersRow, error) {
	rows, err := q.query(ctx, q.getPeersStmt, getPeers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*GetPeersRow
	for rows.Next() {
		var i GetPeersRow
		if err := rows.Scan(&i.Username, &i.PublicKey, &i.Ipv4); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateDevice = `-- name: UpdateDevice :exec
UPDATE devices
SET external_id = ?1, healthy = ?2, last_updated = ?3, last_seen = ?4
WHERE serial = ?5 AND platform = ?6
`

type UpdateDeviceParams struct {
	ExternalID  sql.NullString
	Healthy     bool
	LastUpdated sql.NullString
	LastSeen    sql.NullString
	Serial      string
	Platform    string
}

func (q *Queries) UpdateDevice(ctx context.Context, arg UpdateDeviceParams) error {
	_, err := q.exec(ctx, q.updateDeviceStmt, updateDevice,
		arg.ExternalID,
		arg.Healthy,
		arg.LastUpdated,
		arg.LastSeen,
		arg.Serial,
		arg.Platform,
	)
	return err
}
