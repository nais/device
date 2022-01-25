START TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
-- TODO: change password_hash to NOT NULL in a later migration
ALTER TABLE gateway ADD password_hash VARCHAR(255) NULL;
INSERT INTO migrations (version, created)
VALUES (5, now());
COMMIT;