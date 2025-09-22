-- name: GetApproval :one
SELECT * FROM approvals WHERE user_id = @user_id;

-- name: GetApprovals :many
SELECT * FROM approvals;

-- name: Approve :exec
INSERT INTO approvals (user_id, approved_at) VALUES (@user_id, DATETIME('now')) ON CONFLICT(user_id) DO NOTHING;

-- name: RevokeApproval :exec
DELETE FROM approvals WHERE user_id = @user_id;
