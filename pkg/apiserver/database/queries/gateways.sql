-- name: GetGateways :many
SELECT * FROM gateways;

-- name: GetGatewayAccessGroupIDs :many
SELECT group_id FROM gateway_access_group_ids WHERE gateway_name = ?;

-- name: GetGatewayRoutes :many
SELECT route FROM gateway_routes WHERE gateway_name = ?;

-- name: GetGatewayByName :one
SELECT * FROM gateways WHERE name = ?;

-- name: UpdateGateway :exec
UPDATE gateways
SET public_key = ?, endpoint = ?, ip = ?, requires_privileged_access = ?, password_hash = ?
WHERE name = ?;

-- name: UpdateGatewayDynamicFields :exec
UPDATE gateways
SET requires_privileged_access = ?
WHERE name = ?;

-- name: AddGateway :exec
INSERT INTO gateways (name, endpoint, public_key, ip, password_hash, requires_privileged_access)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT (name) DO
    UPDATE SET endpoint = excluded.endpoint, public_key = excluded.public_key, password_hash = excluded.password_hash;

-- name: DeleteGatewayAccessGroupIDs :exec
DELETE FROM gateway_access_group_ids WHERE gateway_name = ?;

-- name: AddGatewayAccessGroupID :exec
INSERT INTO gateway_access_group_ids (gateway_name, group_id)
VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: DeleteGatewayRoutes :exec
DELETE FROM gateway_routes WHERE gateway_name = ?;

-- name: AddGatewayRoute :exec
INSERT INTO gateway_routes (gateway_name, route)
VALUES (?, ?)
ON CONFLICT DO NOTHING;