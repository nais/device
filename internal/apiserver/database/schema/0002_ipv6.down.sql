ALTER TABLE devices DROP COLUMN ipv6;
ALTER TABLE devices RENAME COLUMN ipv4 TO ip;

ALTER TABLE gateways DROP COLUMN ipv6;
ALTER TABLE gateways RENAME COLUMN ipv4 TO ip;
