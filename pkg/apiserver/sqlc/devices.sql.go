// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0
// source: devices.sql

package sqlc

import (
	"context"
	"database/sql"
)

const addDevice = `-- name: AddDevice :exec
INSERT INTO devices (serial, username, public_key, ip, healthy, platform)
VALUES (?1, ?2, ?3, ?4, ?5, ?6)
ON CONFLICT(serial, platform) DO
    UPDATE SET username = excluded.username, public_key = excluded.public_key
`

type AddDeviceParams struct {
	Serial    string
	Username  string
	PublicKey string
	Ip        string
	Healthy   bool
	Platform  string
}

func (q *Queries) AddDevice(ctx context.Context, arg AddDeviceParams) error {
	_, err := q.exec(ctx, q.addDeviceStmt, addDevice,
		arg.Serial,
		arg.Username,
		arg.PublicKey,
		arg.Ip,
		arg.Healthy,
		arg.Platform,
	)
	return err
}

const getDeviceByID = `-- name: GetDeviceByID :one
SELECT id, username, serial, platform, healthy, last_updated, public_key, ip FROM devices WHERE id = ?1
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
		&i.Ip,
	)
	return &i, err
}

const getDeviceByPublicKey = `-- name: GetDeviceByPublicKey :one
SELECT id, username, serial, platform, healthy, last_updated, public_key, ip FROM devices WHERE public_key = ?1
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
		&i.Ip,
	)
	return &i, err
}

const getDeviceBySerialAndPlatform = `-- name: GetDeviceBySerialAndPlatform :one
SELECT id, username, serial, platform, healthy, last_updated, public_key, ip from devices WHERE serial = ?1 AND platform = ?2
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
		&i.Ip,
	)
	return &i, err
}

const getDevices = `-- name: GetDevices :many
SELECT id, username, serial, platform, healthy, last_updated, public_key, ip FROM devices ORDER BY id
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
			&i.Ip,
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

const updateDevice = `-- name: UpdateDevice :exec
UPDATE devices
SET healthy = ?1, last_updated = ?2
WHERE serial = ?3 AND platform = ?4
`

type UpdateDeviceParams struct {
	Healthy     bool
	LastUpdated sql.NullString
	Serial      string
	Platform    string
}

func (q *Queries) UpdateDevice(ctx context.Context, arg UpdateDeviceParams) error {
	_, err := q.exec(ctx, q.updateDeviceStmt, updateDevice,
		arg.Healthy,
		arg.LastUpdated,
		arg.Serial,
		arg.Platform,
	)
	return err
}
