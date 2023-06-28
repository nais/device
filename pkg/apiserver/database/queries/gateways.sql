-- name: GetGateways :many
SELECT * FROM gateways;

-- name: GetGatewayByName :one
SELECT * FROM gateways WHERE name = $1;

-- name: UpdateGateway :exec
UPDATE gateways
SET public_key = $1, access_group_ids = $2, endpoint = $3, ip = $4, routes = $5, requires_privileged_access = $6, password_hash = $7
WHERE name = $8;

-- name: UpdateGatewayDynamicFields :exec
UPDATE gateways
SET access_group_ids = $1, routes = $2, requires_privileged_access = $3
WHERE name = $4;

-- name: AddGateway :exec
INSERT INTO gateways (name, endpoint, public_key, ip, password_hash, access_group_ids, routes, requires_privileged_access)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (name) DO UPDATE SET endpoint = excluded.endpoint, public_key = excluded.public_key, password_hash = excluded.password_hash;