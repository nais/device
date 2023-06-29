-- name: GetGateways :many
SELECT * FROM gateways;

-- name: GetGatewayAccessGroupIDs :many
SELECT group_id FROM gateway_access_group_ids WHERE gateway_name = $1;

-- name: GetGatewayRoutes :many
SELECT route FROM gateway_routes WHERE gateway_name = $1;

-- name: GetGatewayByName :one
SELECT * FROM gateways WHERE name = $1;

-- name: UpdateGateway :exec
UPDATE gateways
SET public_key = $1, endpoint = $2, ip = $3, requires_privileged_access = $4, password_hash = $5
WHERE name = $6;

-- name: UpdateGatewayDynamicFields :exec
UPDATE gateways
SET requires_privileged_access = $1
WHERE name = $2;

-- name: AddGateway :exec
INSERT INTO gateways (name, endpoint, public_key, ip, password_hash, requires_privileged_access)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (name) DO UPDATE SET endpoint = excluded.endpoint, public_key = excluded.public_key, password_hash = excluded.password_hash;

-- name: DeleteGatewayAccessGroupIDs :exec
DELETE FROM gateway_access_group_ids WHERE gateway_name = $1;

-- name: AddGatewayAccessGroupID :exec
INSERT INTO gateway_access_group_ids (gateway_name, group_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteGatewayRoutes :exec
DELETE FROM gateway_routes WHERE gateway_name = $1;

-- name: AddGatewayRoute :exec
INSERT INTO gateway_routes (gateway_name, route)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;