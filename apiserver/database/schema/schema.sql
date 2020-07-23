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
    id               serial PRIMARY KEY,
    name             varchar,
    access_group_ids varchar,
    endpoint         varchar(21),
    public_key       varchar(44) NOT NULL UNIQUE,
    ip               varchar(15) UNIQUE,
    routes           varchar
);

CREATE TABLE session
(
    key       varchar,
    expiry    bigint,
    device_id integer REFERENCES device (id),
    groups    varchar
);
