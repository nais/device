BEGIN;

ALTER TABLE device RENAME TO devices;
ALTER TABLE gateway RENAME TO gateways;
ALTER TABLE session RENAME TO sessions;

COMMIT;