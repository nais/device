-- Run the entire migration as an atomic operation.
START TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;

-- Indexed timestamps on session.expiry
ALTER TABLE session ADD expiry_ts TIMESTAMP WITH TIME ZONE;
UPDATE session SET expiry_ts = to_timestamp(expiry);
ALTER TABLE session DROP expiry;
ALTER TABLE session RENAME COLUMN expiry_ts TO expiry;
CREATE INDEX expiry_idx ON session (expiry);

-- Non-indexed timestamps on device.last_updated
ALTER TABLE device ADD last_updated_ts TIMESTAMP WITH TIME ZONE;
UPDATE device SET last_updated_ts = to_timestamp(last_updated);
ALTER TABLE device DROP last_updated;
ALTER TABLE device RENAME COLUMN last_updated_ts TO last_updated;

-- Non-indexed timestamps on device.kolide_last_seen
ALTER TABLE device ADD kolide_last_seen_ts TIMESTAMP WITH TIME ZONE;
UPDATE device SET kolide_last_seen_ts = to_timestamp(kolide_last_seen);
ALTER TABLE device DROP kolide_last_seen;
ALTER TABLE device RENAME COLUMN kolide_last_seen_ts TO kolide_last_seen;

-- Mark this database migration as completed.
INSERT INTO migrations (version, created)
VALUES (4, now());
COMMIT;
