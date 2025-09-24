-- name: GetAcceptance :one
SELECT * FROM acceptances WHERE user_id = @user_id;

-- name: GetAcceptances :many
SELECT * FROM acceptances;

-- name: AcceptAcceptableUse :exec
INSERT INTO acceptances (user_id, accepted_at) VALUES (@user_id, @accepted_at) ON CONFLICT(user_id) DO NOTHING;

-- name: RejectAcceptableUse :exec
DELETE FROM acceptances WHERE user_id = @user_id;
