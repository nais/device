-- name: GetSessionByKey :one
SELECT sqlc.embed(s), sqlc.embed(d) FROM sessions s
JOIN devices d ON d.id = s.device_id WHERE s.key = ?;

-- name: GetSessions :many
SELECT sqlc.embed(s), sqlc.embed(d) FROM sessions s
JOIN devices d ON d.id = s.device_id WHERE s.expiry > DATE('now');

-- name: GetMostRecentDeviceSession :one
SELECT sqlc.embed(s), sqlc.embed(d) FROM sessions s
JOIN devices d ON d.id = s.device_id
WHERE s.device_id = ?
ORDER BY s.expiry DESC
LIMIT 1;

-- name: AddSession :exec
INSERT INTO sessions (key, expiry, device_id, object_id)
VALUES (?, ?, ?, ?);

-- name: AddSessionAccessGroupID :exec
INSERT INTO session_access_group_ids (session_key, group_id)
VALUES (?, ?);

-- name: GetSessionGroupIDs :many
SELECT group_id FROM session_access_group_ids WHERE session_key = ?;