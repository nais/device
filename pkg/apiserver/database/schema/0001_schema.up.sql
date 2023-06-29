BEGIN;

-- types

CREATE TYPE platform AS ENUM ('darwin', 'linux', 'windows');

-- tables

CREATE TABLE devices
(
    id SERIAL PRIMARY KEY,
    username VARCHAR,
    serial VARCHAR,
    platform platform NOT NULL,
    healthy BOOLEAN,
    last_updated TIMESTAMP WITH TIME ZONE,
    public_key VARCHAR(44) NOT NULL UNIQUE,
    ip VARCHAR(15) UNIQUE,
    UNIQUE (serial, platform)
);

CREATE TABLE gateways
(
    id SERIAL PRIMARY KEY,
    name VARCHAR NOT NULL UNIQUE,
    access_group_ids VARCHAR DEFAULT '',
    endpoint VARCHAR(21),
    public_key VARCHAR(44) NOT NULL UNIQUE,
    ip VARCHAR(15) UNIQUE,
    routes VARCHAR DEFAULT '',
    requires_privileged_access BOOLEAN DEFAULT FALSE,
    password_hash VARCHAR(255) NULL
);

CREATE TABLE sessions
(
    key VARCHAR UNIQUE,
    expiry TIMESTAMP WITH TIME ZONE,
    device_id INTEGER NOT NULL,
    groups VARCHAR,
    object_id VARCHAR
);

-- indexes

CREATE INDEX ON sessions (expiry);
CREATE INDEX ON devices (LOWER(username));

-- foreign keys

ALTER TABLE sessions
ADD FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE;

COMMIT;