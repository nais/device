-- Run the entire migration as an atomic operation.
START TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;

CREATE TYPE platform AS ENUM ('darwin', 'linux', 'windows');

CREATE TABLE device
(
    id               serial PRIMARY KEY,
    username         varchar,
    serial           varchar,
    psk              varchar(44),
    platform         platform,
    healthy          boolean,
    last_updated     bigint,
    kolide_last_seen bigint,
    public_key       varchar(44) NOT NULL UNIQUE,
    ip               varchar(15) UNIQUE,
    UNIQUE (serial, platform)
);

CREATE TABLE gateway
(
    id                         serial PRIMARY KEY,
    name                       varchar     NOT NULL UNIQUE,
    access_group_ids           varchar DEFAULT '',
    endpoint                   varchar(21),
    public_key                 varchar(44) NOT NULL UNIQUE,
    ip                         varchar(15) UNIQUE,
    routes                     varchar DEFAULT '',
    requires_privileged_access boolean DEFAULT false
);

CREATE TABLE session
(
    key       varchar,
    expiry    bigint,
    device_id integer REFERENCES device (id),
    groups    varchar,
    object_id varchar
);

-- Database migration
CREATE TABLE migrations
(
    version int primary key          not null,
    created timestamp with time zone not null
);

-- Mark this database migration as completed.
INSERT INTO migrations (version, created)
VALUES (1, now());
COMMIT;
