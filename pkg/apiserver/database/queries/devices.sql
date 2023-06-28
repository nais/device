-- name: GetDevices :many
SELECT * FROM devices;

-- name: GetDeviceByPublicKey :one
SELECT * FROM devices WHERE public_key = $1;

-- name: GetDeviceByID :one
SELECT * FROM devices WHERE id = $1;

-- name: GetDeviceBySerialAndPlatform :one
SELECT * from devices WHERE serial = $1 AND platform = $2;

-- name: UpdateDevice :exec
UPDATE devices
SET healthy = $1, kolide_last_seen = $2, last_updated = NOW()
WHERE serial = $3 AND platform = $4;

-- name: AddDevice :exec
INSERT INTO devices (serial, username, public_key, ip, healthy, psk, platform)
VALUES ($1, $2, $3, $4, $5, '', $6)
ON CONFLICT(serial, platform) DO
    UPDATE SET username = excluded.username, public_key = excluded.public_key;