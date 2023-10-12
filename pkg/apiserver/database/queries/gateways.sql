-- name: GetGateways :many
SELECT * FROM gateways ORDER BY name;

-- name: GetGatewayAccessGroupIDs :many
SELECT group_id FROM gateway_access_group_ids WHERE gateway_name = @gateway_name ORDER BY group_id;

-- name: GetGatewayRoutes :many
SELECT route FROM gateway_routes WHERE gateway_name = @gateway_name ORDER BY route;

-- name: GetGatewayByName :one
SELECT * FROM gateways WHERE name = @name;

-- name: UpdateGateway :exec
UPDATE gateways
SET public_key = @public_key, endpoint = @endpoint, ipv4 = @ipv4, ipv6 = @ipv6, requires_privileged_access = @requires_privileged_access, password_hash = @password_hash
WHERE name = @name;

-- name: UpdateGatewayDynamicFields :exec
UPDATE gateways
SET requires_privileged_access = @requires_privileged_access
WHERE name = @name;

-- name: AddGateway :exec
INSERT INTO gateways (name, endpoint, public_key, ipv4, ipv6, password_hash, requires_privileged_access)
VALUES (@name, @endpoint, @public_key, @ipv4, @ipv6, @password_hash, @requires_privileged_access)
ON CONFLICT (name) DO
    UPDATE SET endpoint = excluded.endpoint, public_key = excluded.public_key, password_hash = excluded.password_hash, ipv6 = excluded.ipv6;

-- name: DeleteGatewayAccessGroupIDs :exec
DELETE FROM gateway_access_group_ids WHERE gateway_name = @gateway_name;

-- name: AddGatewayAccessGroupID :exec
INSERT INTO gateway_access_group_ids (gateway_name, group_id)
VALUES (@gateway_name, @group_id)
ON CONFLICT DO NOTHING;

-- name: DeleteGatewayRoutes :exec
DELETE FROM gateway_routes WHERE gateway_name = @gateway_name;

-- name: AddGatewayRoute :exec
INSERT INTO gateway_routes (gateway_name, route)
VALUES (@gateway_name, @route)
ON CONFLICT DO NOTHING;
