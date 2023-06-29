-- name: GetSessionByKey :one
SELECT sqlc.embed(s), sqlc.embed(d) FROM sessions s
JOIN devices d ON d.id = s.device_id WHERE s.key = $1;

-- name: GetSessions :many
SELECT sqlc.embed(s), sqlc.embed(d) FROM sessions s
JOIN devices d ON d.id = s.device_id WHERE s.expiry > NOW();

-- name: GetMostRecentDeviceSession :one
SELECT sqlc.embed(s), sqlc.embed(d) FROM sessions s
JOIN devices d ON d.id = s.device_id
WHERE s.device_id = $1
ORDER BY s.expiry DESC
LIMIT 1;

-- name: AddSession :exec
INSERT INTO sessions (key, expiry, device_id, object_id)
VALUES ($1, $2, $3, $4);

-- name: AddSessionAccessGroupID :exec
INSERT INTO session_access_group_ids (session_key, group_id)
VALUES ($1, $2);

-- name: GetSessionGroupIDs :many
SELECT group_id FROM session_access_group_ids WHERE session_key = $1;