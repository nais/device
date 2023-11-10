ALTER TABLE devices ADD COLUMN ipv6 TEXT NOT NULL DEFAULT '';
ALTER TABLE devices RENAME COLUMN ip TO ipv4;
CREATE INDEX devices_ipv6_idx ON devices (ipv6);

ALTER TABLE gateways ADD COLUMN ipv6 TEXT NOT NULL DEFAULT '';
ALTER TABLE gateways RENAME COLUMN ip TO ipv4;
CREATE INDEX gateways_ipv6_idx ON gateways (ipv6);
