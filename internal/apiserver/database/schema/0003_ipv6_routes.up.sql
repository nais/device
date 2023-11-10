ALTER TABLE gateway_routes ADD COLUMN family TEXT CHECK(family IN ('IPv4', 'IPv6')) NOT NULL DEFAULT 'IPv4';
