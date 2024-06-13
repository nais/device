// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: sessions.sql

package sqlc

import (
	"context"
)

const addSession = `-- name: AddSession :exec
INSERT INTO sessions (key, expiry, device_id, object_id)
VALUES (?1, ?2, ?3, ?4)
`

type AddSessionParams struct {
	Key      string
	Expiry   string
	DeviceID int64
	ObjectID string
}

func (q *Queries) AddSession(ctx context.Context, arg AddSessionParams) error {
	_, err := q.exec(ctx, q.addSessionStmt, addSession,
		arg.Key,
		arg.Expiry,
		arg.DeviceID,
		arg.ObjectID,
	)
	return err
}

const addSessionAccessGroupID = `-- name: AddSessionAccessGroupID :exec
INSERT INTO session_access_group_ids (session_key, group_id)
VALUES (?1, ?2)
`

type AddSessionAccessGroupIDParams struct {
	SessionKey string
	GroupID    string
}

func (q *Queries) AddSessionAccessGroupID(ctx context.Context, arg AddSessionAccessGroupIDParams) error {
	_, err := q.exec(ctx, q.addSessionAccessGroupIDStmt, addSessionAccessGroupID, arg.SessionKey, arg.GroupID)
	return err
}

const getMostRecentDeviceSession = `-- name: GetMostRecentDeviceSession :one
SELECT s."key", s.expiry, s.device_id, s.object_id, d.id, d.username, d.serial, d.platform, d.healthy, d.last_updated, d.public_key, d.ipv4, d.ipv6, d.last_seen, d.issues, d.external_id FROM sessions s
JOIN devices d ON d.id = s.device_id
WHERE s.device_id = ?1
ORDER BY s.expiry DESC
LIMIT 1
`

type GetMostRecentDeviceSessionRow struct {
	Session Session
	Device  Device
}

func (q *Queries) GetMostRecentDeviceSession(ctx context.Context, sessionDeviceID int64) (*GetMostRecentDeviceSessionRow, error) {
	row := q.queryRow(ctx, q.getMostRecentDeviceSessionStmt, getMostRecentDeviceSession, sessionDeviceID)
	var i GetMostRecentDeviceSessionRow
	err := row.Scan(
		&i.Session.Key,
		&i.Session.Expiry,
		&i.Session.DeviceID,
		&i.Session.ObjectID,
		&i.Device.ID,
		&i.Device.Username,
		&i.Device.Serial,
		&i.Device.Platform,
		&i.Device.Healthy,
		&i.Device.LastUpdated,
		&i.Device.PublicKey,
		&i.Device.Ipv4,
		&i.Device.Ipv6,
		&i.Device.LastSeen,
		&i.Device.Issues,
		&i.Device.ExternalID,
	)
	return &i, err
}

const getSessionByKey = `-- name: GetSessionByKey :one
SELECT s."key", s.expiry, s.device_id, s.object_id, d.id, d.username, d.serial, d.platform, d.healthy, d.last_updated, d.public_key, d.ipv4, d.ipv6, d.last_seen, d.issues, d.external_id FROM sessions s
JOIN devices d ON d.id = s.device_id WHERE s.key = ?1
`

type GetSessionByKeyRow struct {
	Session Session
	Device  Device
}

func (q *Queries) GetSessionByKey(ctx context.Context, sessionKey string) (*GetSessionByKeyRow, error) {
	row := q.queryRow(ctx, q.getSessionByKeyStmt, getSessionByKey, sessionKey)
	var i GetSessionByKeyRow
	err := row.Scan(
		&i.Session.Key,
		&i.Session.Expiry,
		&i.Session.DeviceID,
		&i.Session.ObjectID,
		&i.Device.ID,
		&i.Device.Username,
		&i.Device.Serial,
		&i.Device.Platform,
		&i.Device.Healthy,
		&i.Device.LastUpdated,
		&i.Device.PublicKey,
		&i.Device.Ipv4,
		&i.Device.Ipv6,
		&i.Device.LastSeen,
		&i.Device.Issues,
		&i.Device.ExternalID,
	)
	return &i, err
}

const getSessionGroupIDs = `-- name: GetSessionGroupIDs :many
SELECT group_id FROM session_access_group_ids WHERE session_key = ?1 ORDER BY group_id
`

func (q *Queries) GetSessionGroupIDs(ctx context.Context, sessionKey string) ([]string, error) {
	rows, err := q.query(ctx, q.getSessionGroupIDsStmt, getSessionGroupIDs, sessionKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var group_id string
		if err := rows.Scan(&group_id); err != nil {
			return nil, err
		}
		items = append(items, group_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getSessions = `-- name: GetSessions :many
SELECT s."key", s.expiry, s.device_id, s.object_id, d.id, d.username, d.serial, d.platform, d.healthy, d.last_updated, d.public_key, d.ipv4, d.ipv6, d.last_seen, d.issues, d.external_id FROM sessions s
JOIN devices d ON d.id = s.device_id
ORDER BY s.expiry
`

type GetSessionsRow struct {
	Session Session
	Device  Device
}

func (q *Queries) GetSessions(ctx context.Context) ([]*GetSessionsRow, error) {
	rows, err := q.query(ctx, q.getSessionsStmt, getSessions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*GetSessionsRow
	for rows.Next() {
		var i GetSessionsRow
		if err := rows.Scan(
			&i.Session.Key,
			&i.Session.Expiry,
			&i.Session.DeviceID,
			&i.Session.ObjectID,
			&i.Device.ID,
			&i.Device.Username,
			&i.Device.Serial,
			&i.Device.Platform,
			&i.Device.Healthy,
			&i.Device.LastUpdated,
			&i.Device.PublicKey,
			&i.Device.Ipv4,
			&i.Device.Ipv6,
			&i.Device.LastSeen,
			&i.Device.Issues,
			&i.Device.ExternalID,
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

const removeExpiredSessions = `-- name: RemoveExpiredSessions :exec
DELETE FROM sessions WHERE DATETIME(expiry) < DATETIME('now')
`

func (q *Queries) RemoveExpiredSessions(ctx context.Context) error {
	_, err := q.exec(ctx, q.removeExpiredSessionsStmt, removeExpiredSessions)
	return err
}
