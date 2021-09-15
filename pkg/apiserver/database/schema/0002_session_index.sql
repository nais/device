CREATE UNIQUE INDEX session_key_idx ON session (key);
INSERT INTO migrations (version, created)
VALUES (2, now());