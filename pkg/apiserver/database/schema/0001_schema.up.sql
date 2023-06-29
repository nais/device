BEGIN;

-- types

CREATE TYPE platform AS ENUM ('darwin', 'linux', 'windows');

-- tables

CREATE TABLE devices
(
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    serial TEXT NOT NULL,
    platform platform NOT NULL,
    healthy BOOLEAN NOT NULL DEFAULT false,
    last_updated TIMESTAMP WITH TIME ZONE,
    public_key TEXT NOT NULL,
    ip TEXT NOT NULL,
    UNIQUE (serial, platform),
    UNIQUE (public_key),
    UNIQUE (ip)
);

CREATE TABLE gateways
(
    name TEXT PRIMARY KEY,
    endpoint TEXT NOT NULL,
    public_key TEXT NOT NULL,
    ip TEXT NOT NULL,
    requires_privileged_access BOOLEAN NOT NULL DEFAULT false,
    password_hash TEXT NOT NULL,
    UNIQUE (public_key),
    UNIQUE (ip)
);

CREATE TABLE gateway_access_group_ids
(
    gateway_name TEXT NOT NULL,
    group_id TEXT NOT NULL,
    PRIMARY KEY(gateway_name, group_id)
);

CREATE TABLE gateway_routes
(
    gateway_name TEXT NOT NULL,
    route TEXT NOT NULL,
    PRIMARY KEY(gateway_name, route)
);

CREATE TABLE sessions
(
    key TEXT NOT NULL,
    expiry TIMESTAMP WITH TIME ZONE NOT NULL,
    device_id INTEGER NOT NULL,
    object_id TEXT NOT NULL,
    UNIQUE (key)
);

CREATE TABLE session_access_group_ids (
    session_key TEXT NOT NULL,
    group_id TEXT NOT NULL,
    PRIMARY KEY(session_key, group_id)
);

-- indexes

CREATE INDEX ON sessions (expiry);
CREATE INDEX ON devices (LOWER(username));

-- foreign keys

ALTER TABLE sessions
ADD FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE;

ALTER TABLE gateway_access_group_ids
ADD FOREIGN KEY (gateway_name) REFERENCES gateways(name) ON DELETE CASCADE;

ALTER TABLE gateway_routes
ADD FOREIGN KEY (gateway_name) REFERENCES gateways(name) ON DELETE CASCADE;

ALTER TABLE session_access_group_ids
ADD FOREIGN KEY (session_key) REFERENCES sessions(key) ON DELETE CASCADE;

COMMIT;