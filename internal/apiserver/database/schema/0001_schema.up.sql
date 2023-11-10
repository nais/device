-- BEGIN; # golang migrate automatically wraps migrations in a transaction

-- tables

CREATE TABLE devices
(
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL,
    serial TEXT NOT NULL,
    platform TEXT CHECK(platform IN ('darwin', 'linux', 'windows')) NOT NULL,
    healthy BOOLEAN NOT NULL DEFAULT 0,
    last_updated TEXT,
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
    requires_privileged_access BOOLEAN NOT NULL DEFAULT 0,
    password_hash TEXT NOT NULL,
    UNIQUE (public_key),
    UNIQUE (ip)
);

CREATE TABLE gateway_access_group_ids
(
    gateway_name TEXT NOT NULL,
    group_id TEXT NOT NULL,
    PRIMARY KEY(gateway_name, group_id),
    FOREIGN KEY (gateway_name) REFERENCES gateways(name) ON DELETE CASCADE
);

CREATE TABLE gateway_routes
(
    gateway_name TEXT NOT NULL,
    route TEXT NOT NULL,
    PRIMARY KEY(gateway_name, route),
    FOREIGN KEY (gateway_name) REFERENCES gateways(name) ON DELETE CASCADE
);

CREATE TABLE sessions
(
    key TEXT NOT NULL,
    expiry TEXT NOT NULL,
    device_id INTEGER NOT NULL,
    object_id TEXT NOT NULL,
    UNIQUE (key),
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

CREATE TABLE session_access_group_ids (
    session_key TEXT NOT NULL,
    group_id TEXT NOT NULL,
    PRIMARY KEY(session_key, group_id),
    FOREIGN KEY (session_key) REFERENCES sessions(key) ON DELETE CASCADE
);

-- indexes

CREATE INDEX session_expiry_idx ON sessions (expiry);
CREATE INDEX devices_username_idx ON devices (LOWER(username));

-- COMMIT; # golang migrate automatically wraps migrations in a transaction
