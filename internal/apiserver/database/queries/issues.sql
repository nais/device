-- name: SetKolideIssue :exec
INSERT INTO kolide_issues (id, device_id, check_id, title, detected_at, resolved_at, last_updated, ignored)
VALUES (@id, @device_id, @check_id, @title, @detected_at, @resolved_at, @last_updated, @ignored)
	ON CONFLICT(id) DO UPDATE SET 
	device_id = excluded.device_id,
	check_id = excluded.check_id,
	title = excluded.title,
	detected_at = excluded.detected_at,
	resolved_at = excluded.resolved_at,
	last_updated = excluded.last_updated,
	ignored = excluded.ignored;

-- name: GetKolideIssuesForDevice :many
SELECT kolide_issues.*, kolide_checks.*
FROM kolide_issues
JOIN kolide_checks ON kolide_checks.id = kolide_issues.check_id
WHERE kolide_issues.device_id = @device_id;

-- name: GetKolideIssues :many
SELECT kolide_issues.*, kolide_checks.*
FROM kolide_issues
JOIN kolide_checks ON kolide_checks.id = kolide_issues.check_id;

-- name: SetKolideCheck :exec
INSERT INTO kolide_checks (id, tags, display_name, description)
VALUES (@id, @tags, @display_name, @description) 
	ON CONFLICT(id) DO UPDATE SET 
	tags = excluded.tags,
	display_name = excluded.display_name,
	description = excluded.description;

-- name: GetKolideChecks :many
SELECT * FROM kolide_checks;

-- name: GetKolideCheck :one
SELECT * FROM kolide_checks WHERE id = @id;

-- name: TruncateKolideIssues :exec
DELETE FROM kolide_issues;

-- name: DeleteKolideIssuesForDevice :exec
DELETE FROM kolide_issues WHERE device_id = @device_id;
