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
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    access_group_ids TEXT NOT NULL DEFAULT '',
    endpoint TEXT NOT NULL,
    public_key TEXT NOT NULL,
    ip TEXT NOT NULL,
    routes TEXT NOT NULL DEFAULT '',
    requires_privileged_access BOOLEAN NOT NULL DEFAULT false,
    password_hash TEXT NOT NULL,
    UNIQUE (name),
    UNIQUE (public_key),
    UNIQUE (ip)
);

CREATE TABLE sessions
(
    key TEXT NOT NULL,
    expiry TIMESTAMP WITH TIME ZONE NOT NULL,
    device_id INTEGER NOT NULL,
    groups TEXT NOT NULL DEFAULT '',
    object_id TEXT NOT NULL,
    UNIQUE (key)
);

-- indexes

CREATE INDEX ON sessions (expiry);
CREATE INDEX ON devices (LOWER(username));

-- foreign keys

ALTER TABLE sessions
ADD FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE;

COMMIT;