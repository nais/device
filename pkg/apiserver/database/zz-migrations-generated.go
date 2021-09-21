// file generated by go generate

package database

var migrations = []string{
"-- Run the entire migration as an atomic operation.\nSTART TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;\n\nCREATE TYPE platform AS ENUM ('darwin', 'linux', 'windows');\n\nCREATE TABLE device\n(\n    id               serial PRIMARY KEY,\n    username         varchar,\n    serial           varchar,\n    psk              varchar(44),\n    platform         platform,\n    healthy          boolean,\n    last_updated     bigint,\n    kolide_last_seen bigint,\n    public_key       varchar(44) NOT NULL UNIQUE,\n    ip               varchar(15) UNIQUE,\n    UNIQUE (serial, platform)\n);\n\nCREATE TABLE gateway\n(\n    id                         serial PRIMARY KEY,\n    name                       varchar     NOT NULL UNIQUE,\n    access_group_ids           varchar DEFAULT '',\n    endpoint                   varchar(21),\n    public_key                 varchar(44) NOT NULL UNIQUE,\n    ip                         varchar(15) UNIQUE,\n    routes                     varchar DEFAULT '',\n    requires_privileged_access boolean DEFAULT false\n);\n\nCREATE TABLE session\n(\n    key       varchar,\n    expiry    bigint,\n    device_id integer REFERENCES device (id),\n    groups    varchar,\n    object_id varchar\n);\n\n-- Database migration\nCREATE TABLE migrations\n(\n    version int primary key          not null,\n    created timestamp with time zone not null\n);\n\n-- Mark this database migration as completed.\nINSERT INTO migrations (version, created)\nVALUES (1, now());\nCOMMIT;\n",
"CREATE UNIQUE INDEX session_key_idx ON session (key);\nINSERT INTO migrations (version, created)\nVALUES (2, now());",
"CREATE INDEX device_lower_case_username ON device ((lower(username)));\nINSERT INTO migrations (version, created)\nVALUES (3, now());",
"-- Run the entire migration as an atomic operation.\nSTART TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;\n\n-- Indexed timestamps on session.expiry\nALTER TABLE session ADD expiry_ts TIMESTAMP WITH TIME ZONE;\nUPDATE session SET expiry_ts = to_timestamp(expiry);\nALTER TABLE session DROP expiry;\nALTER TABLE session RENAME COLUMN expiry_ts TO expiry;\nCREATE INDEX expiry_idx ON session (expiry);\n\n-- Non-indexed timestamps on device.last_updated\nALTER TABLE device ADD last_updated_ts TIMESTAMP WITH TIME ZONE;\nUPDATE device SET last_updated_ts = to_timestamp(last_updated);\nALTER TABLE device DROP last_updated;\nALTER TABLE device RENAME COLUMN last_updated_ts TO last_updated;\n\n-- Non-indexed timestamps on device.kolide_last_seen\nALTER TABLE device ADD kolide_last_seen_ts TIMESTAMP WITH TIME ZONE;\nUPDATE device SET kolide_last_seen_ts = to_timestamp(kolide_last_seen);\nALTER TABLE device DROP kolide_last_seen;\nALTER TABLE device RENAME COLUMN kolide_last_seen_ts TO kolide_last_seen;\n\n-- Mark this database migration as completed.\nINSERT INTO migrations (version, created)\nVALUES (4, now());\nCOMMIT;\n",

}
