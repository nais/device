CREATE TABLE peer
(
    id         serial PRIMARY KEY,
    public_key varchar(44) NOT NULL UNIQUE,
    ip         varchar(15) UNIQUE,
    type       varchar(7)
);

CREATE TABLE client
(
    id         serial PRIMARY KEY,
    serial     varchar(255) UNIQUE,
    psk        varchar(44),
    healthy    boolean,
    last_check timestamp
);

CREATE TABLE gateway
(
    id              serial PRIMARY KEY,
    access_group_id varchar(255),
    endpoint        varchar(21)
);

CREATE TABLE routes
(
    gateway_id INTEGER REFERENCES gateway (id),
    cidr       varchar(18)
);

CREATE TABLE client_peer
(
    client_id INTEGER REFERENCES client (id),
    peer_id   INTEGER REFERENCES peer (id) UNIQUE
);

CREATE TABLE gateway_peer
(
    gateway_id INTEGER REFERENCES gateway (id),
    peer_id    INTEGER REFERENCES peer (id) UNIQUE
);
