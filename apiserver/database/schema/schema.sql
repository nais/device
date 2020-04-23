CREATE TABLE device
(
    id         serial PRIMARY KEY,
    username   varchar,
    serial     varchar UNIQUE,
    psk        varchar(44),
    healthy    boolean,
    last_check timestamp,
    public_key varchar(44) NOT NULL UNIQUE,
    ip         varchar(15) UNIQUE
);

CREATE TABLE gateway
(
    id              serial PRIMARY KEY,
    name            varchar,
    access_group_id varchar,
    endpoint        varchar(21),
    public_key      varchar(44) NOT NULL UNIQUE,
    ip              varchar(15) UNIQUE,
    routes          varchar
);
