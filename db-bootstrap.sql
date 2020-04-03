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

/* apiserver */
INSERT INTO peer (public_key, ip, type)
VALUES ('FUwVtyvs8nIRx9RpUUEopkfV8idmHz9g9K/vf9MFOXI=', '10.255.240.1', 'control');

/* vegar */
BEGIN;
WITH client_key AS
         (INSERT INTO client (serial, psk, healthy) VALUES ('serial1', 'psk1', true) RETURNING id),
     peer_control_key
         AS (INSERT INTO peer (public_key, ip, type) VALUES ('EatjldYVvB91aep5kxDnYsQ37Ufk92IBBIcfma1fzAs=',
                                                             '10.255.240.2', 'control') RETURNING id),
     peer_data_key
         AS (INSERT INTO peer (public_key, ip, type) VALUES ('EatjldYVvB91aep5kxAnYsQ37Ufk92IBBIcfma1fzAA=',
                                                             '10.255.248.2', 'data') RETURNING id)
INSERT
INTO client_peer(client_id, peer_id)
    (SELECT client_key.id, peer.id
     FROM client_key,
          (SELECT id FROM peer_control_key UNION SELECT id FROM peer_data_key) AS peer);

/* johnny */
WITH client_key AS
         (INSERT INTO client (serial, psk, healthy) VALUES ('serial2', 'psk2', true) RETURNING id),
     peer_control_key
         AS (INSERT INTO peer (public_key, ip, type) VALUES ('EatjldYVvB91aep5kxDnYsQ37Ufk92IBBIcfma1fzAA=',
                                                             '10.255.240.3', 'control') RETURNING id),
     peer_data_key
         AS (INSERT INTO peer (public_key, ip, type) VALUES ('EatjldYVvB91aep5kxDnYsa37Ufk92IBBIcfma1fzAA=',
                                                             '10.255.248.3', 'data') RETURNING id)
INSERT
INTO client_peer(client_id, peer_id)
    (SELECT client_key.id, peer.id
     FROM client_key,
          (SELECT id FROM peer_control_key UNION SELECT id FROM peer_data_key) AS peer);


/* gateway 1 */
WITH gateway_key
         AS (INSERT INTO gateway (access_group_id, endpoint) VALUES ('1234-asdf-aad1', '35.228.118.232:51820') RETURNING id),
     peer_control_key
         AS (INSERT INTO peer (public_key, ip, type) VALUES ('QFwvy4pUYXpYm4z9iXw1GZRgjp3iU+3Hsu0UUvre9FM=',
                                                             '10.255.240.4', 'control') RETURNING id),
     peer_data_key
         AS (INSERT INTO peer (public_key, ip, type) VALUES ('55h6JA2ZMPzaoa+iZU62JmqmtgK3ydj4YdT9HkkhnEQ=',
                                                             '10.255.248.4', 'data') RETURNING id)
INSERT
INTO gateway_peer(gateway_id, peer_id)
    (SELECT gateway_key.id, peer.id
     FROM gateway_key,
          (SELECT id FROM peer_control_key UNION SELECT id FROM peer_data_key) AS peer);

/* gateway 2 */
WITH gateway_key
         AS (INSERT INTO gateway (access_group_id, endpoint) VALUES ('1234-asdf-aad2', '35.228.118.232:51820') RETURNING id),
     peer_control_key
         AS (INSERT INTO peer (public_key, ip, type) VALUES ('Whbuh2+T8/m1kJTtByfYQvlD/Efv4xxX9rbe9B2SK2M=',
                                                             '10.255.240.5', 'control') RETURNING id),
     peer_data_key AS
         (INSERT INTO peer (public_key, ip, type) VALUES ('i5AmQLLlPa4fQmfuHj7COCFwmwegI39WMfs/LIdzbFo=',
                                                          '10.255.248.5', 'data') RETURNING id)
INSERT
INTO gateway_peer(gateway_id, peer_id)
    (SELECT gateway_key.id, peer.id
     FROM gateway_key,
          (SELECT id FROM peer_control_key UNION SELECT id FROM peer_data_key) AS peer);

END;
