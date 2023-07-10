-- name: GetDevices :many
SELECT * FROM devices;

-- name: GetDeviceByPublicKey :one
SELECT * FROM devices WHERE public_key = ?;

-- name: GetDeviceByID :one
SELECT * FROM devices WHERE id = ?;

-- name: GetDeviceBySerialAndPlatform :one
SELECT * from devices WHERE serial = ? AND platform = ?;

-- name: UpdateDevice :exec
UPDATE devices
SET healthy = ?, last_updated = DATE('now')
WHERE serial = ? AND platform = ?;

-- name: AddDevice :exec
INSERT INTO devices (serial, username, public_key, ip, healthy, platform)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(serial, platform) DO
    UPDATE SET username = excluded.username, public_key = excluded.public_key;