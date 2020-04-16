CREATE TABLE device
(
    id         serial PRIMARY KEY,
    username   varchar(255),
    serial     varchar(255) UNIQUE,
    psk        varchar(44),
    healthy    boolean,
    last_check timestamp,
    public_key varchar(44) NOT NULL UNIQUE,
    ip         varchar(15) UNIQUE
);

CREATE TABLE gateway
(
    id              serial PRIMARY KEY,
    access_group_id varchar(255),
    endpoint        varchar(21),
    public_key      varchar(44) NOT NULL UNIQUE,
    ip              varchar(15) UNIQUE
);

CREATE TABLE routes
(
    gateway_id INTEGER REFERENCES gateway (id),
    cidr       varchar(18)
);

BEGIN;

INSERT INTO device (serial, username, psk, healthy, public_key, ip)
VALUES ('serial1', 'vegar.sechmann.molvig@nav.no', 'psk1', true, 'EatjldYVvB91aep5kxDnYsQ37Ufk92IBBIcfma1fzAs=',
        '10.255.240.2');

INSERT INTO device (serial, username, psk, healthy, public_key, ip)
VALUES ('serial2', 'johnny.horvi@nav.no', 'psk2', true, 'EatjldYVvB91aep5kxDnYsQ37Ufk92IBBIcfma1fzAA=', '10.255.240.3');

/* gateway 1 */
INSERT INTO gateway (public_key, ip, endpoint)
VALUES ('QFwvy4pUYXpYm4z9iXw1GZRgjp3iU+3Hsu0UUvre9FM=', '10.255.240.4', '35.228.118.232:51820');

/* gateway 2 */
INSERT INTO gateway (public_key, ip, endpoint)
VALUES ('Whbuh2+T8/m1kJTtByfYQvlD/Efv4xxX9rbe9B2SK2M=', '10.255.240.5', '35.228.118.232:51820');

END;
