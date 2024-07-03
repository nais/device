-- Delete all sessions for all device_id's, except the most recent session for each user.
DELETE FROM sessions AS s
WHERE key !=
    (SELECT key FROM sessions
    WHERE device_id = s.device_id
    ORDER BY expiry DESC
    LIMIT 1)
;

CREATE UNIQUE INDEX sessions_device_id_unique ON sessions (device_id);