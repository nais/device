CREATE TABLE peer (
  id serial PRIMARY KEY,
  public_key varchar(44) NOT NULL UNIQUE,
  kind integer NOT NULL
);

CREATE TABLE client (
  peer_id INTEGER REFERENCES peer(id),
  serial varchar(255) UNIQUE,
  psk varchar(44),
  healthy boolean,
  last_check timestamp
);

CREATE TABLE gateway (
  peer_id INTEGER REFERENCES peer(id),
  id serial PRIMARY KEY,
  access_group_id varchar(255),
  endpoint varchar(21)
);

CREATE TABLE routes (
  gateway_id INTEGER REFERENCES gateway(id),
  cidr varchar(18)
);

CREATE TABLE ip (
  peer_id INTEGER REFERENCES peer(id),
  ip varchar(15) UNIQUE
);


