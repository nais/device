-- name: GetDevices :many
SELECT * FROM devices ORDER BY id;

-- name: GetDeviceByPublicKey :one
SELECT * FROM devices WHERE public_key = @public_key;

-- name: GetDeviceByID :one
SELECT * FROM devices WHERE id = @id;

-- name: GetDeviceBySerialAndPlatform :one
SELECT * from devices WHERE serial = @serial AND platform = @platform;

-- name: UpdateDevice :exec
UPDATE devices
SET healthy = @healthy, last_updated = DATE('now')
WHERE serial = @serial AND platform = @platform;

-- name: AddDevice :exec
INSERT INTO devices (serial, username, public_key, ip, healthy, platform)
VALUES (@serial, @username, @public_key, @ip, @healthy, @platform)
ON CONFLICT(serial, platform) DO
    UPDATE SET username = excluded.username, public_key = excluded.public_key;
