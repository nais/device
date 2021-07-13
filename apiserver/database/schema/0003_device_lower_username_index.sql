CREATE INDEX device_lower_case_username ON device ((lower(username)));
INSERT INTO migrations (version, created)
VALUES (3, now());