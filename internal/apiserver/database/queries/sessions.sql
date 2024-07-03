-- name: GetSessionByKey :one
SELECT sqlc.embed(s), sqlc.embed(d) FROM sessions s
JOIN devices d ON d.id = s.device_id WHERE s.key = @session_key;

-- name: GetSessions :many
SELECT sqlc.embed(s), sqlc.embed(d) FROM sessions s
JOIN devices d ON d.id = s.device_id
ORDER BY s.expiry;

-- name: GetMostRecentDeviceSession :one
SELECT sqlc.embed(s), sqlc.embed(d) FROM sessions s
JOIN devices d ON d.id = s.device_id
WHERE s.device_id = @session_device_id
ORDER BY s.expiry DESC
LIMIT 1;

-- name: AddSession :exec
INSERT INTO sessions (key, expiry, device_id, object_id)
VALUES (@key, @expiry, @device_id, @object_id)
ON CONFLICT (device_id) DO UPDATE
SET key = excluded.key,
    expiry = excluded.expiry,
    object_id = excluded.object_id;

-- name: AddSessionAccessGroupID :exec
INSERT INTO session_access_group_ids (session_key, group_id)
VALUES (@session_key, @group_id);

-- name: GetSessionGroupIDs :many
SELECT group_id FROM session_access_group_ids WHERE session_key = @session_key ORDER BY group_id;

-- name: RemoveExpiredSessions :exec
DELETE FROM sessions WHERE DATETIME(expiry) < DATETIME('now');
