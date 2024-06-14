-- name: GetDevices :many
SELECT * FROM devices ORDER BY id;

-- name: GetDeviceByPublicKey :one
SELECT * FROM devices WHERE public_key = @public_key;

-- name: GetDeviceByExternalID :one
SELECT * FROM devices WHERE external_id = @external_id;

-- name: GetDeviceByID :one
SELECT * FROM devices WHERE id = @id;

-- name: GetDeviceBySerialAndPlatform :one
SELECT * from devices WHERE serial = @serial AND platform = @platform;

-- name: UpdateDevice :one
UPDATE devices
SET external_id = @external_id, healthy = @healthy, last_updated = @last_updated, last_seen = @last_seen, issues = @issues
WHERE serial = @serial AND platform = @platform
RETURNING id;
;

-- name: AddDevice :exec
INSERT INTO devices (serial, username, public_key, ipv4, ipv6, healthy, platform)
VALUES (@serial, @username, @public_key, @ipv4, @ipv6, @healthy, @platform)
ON CONFLICT(serial, platform) DO
    UPDATE SET username = excluded.username, public_key = excluded.public_key, ipv6 = excluded.ipv6;

-- name: ClearDeviceIssuesExceptFor :exec
UPDATE devices
SET issues = NULL
WHERE id NOT IN (CAST(@unhealthy_device_ids AS INTEGER[]));
