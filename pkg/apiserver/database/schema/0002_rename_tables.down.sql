BEGIN;

ALTER TABLE devices RENAME TO device;
ALTER TABLE gateways RENAME TO gateway;
ALTER TABLE sessions RENAME TO session;

COMMIT;