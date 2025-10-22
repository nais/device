-- name: GetGatewayJitaGrantsForUser :many
SELECT * FROM gateway_jita_grants
WHERE user_id = @user_id
ORDER BY id DESC
LIMIT 10;

-- name: UserHasAccessToPrivilegedGateway :one
SELECT EXISTS(
    SELECT id FROM gateway_jita_grants
    WHERE
        user_id = @user_id
        AND gateway_name = @gateway_name
        AND DATETIME(expires) > DATETIME('now')
        AND revoked IS NULL
) AS has_access;

-- name: UsersWithAccessToPrivilegedGateway :many
SELECT user_id FROM gateway_jita_grants
WHERE
    gateway_name = @gateway_name
    AND DATETIME(expires) > DATETIME('now')
    AND revoked IS NULL;

-- name: GrantPrivilegedGatewayAccess :exec
INSERT INTO gateway_jita_grants (
    user_id,
    gateway_name,
    created,
    expires,
    reason
)
VALUES (
    @user_id,
    @gateway_name,
    @created,
    @expires,
    @reason
);

-- name: RevokePrivilegedGatewayAccess :exec
UPDATE gateway_jita_grants
SET revoked = @revoked
WHERE
    user_id = @user_id
    AND gateway_name = @gateway_name
    AND revoked IS NULL;
