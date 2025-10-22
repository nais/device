CREATE TABLE gateway_jita_grants (
    id INTEGER PRIMARY KEY,
    user_id TEXT NOT NULL,
    gateway_name TEXT NOT NULL,
    created TEXT NOT NULL,
    expires TEXT NOT NULL,
    revoked TEXT,
    reason TEXT NOT NULL,
    FOREIGN KEY (gateway_name) REFERENCES gateways(name) ON DELETE CASCADE
);

CREATE INDEX gateway_jita_grants_user_id_idx ON gateway_jita_grants (user_id);
CREATE INDEX gateway_jita_grants_gateway_idx ON gateway_jita_grants (gateway_name);
CREATE INDEX gateway_jita_grants_expires_idx ON gateway_jita_grants (expires);
CREATE INDEX gateway_jita_grants_revoked_idx ON gateway_jita_grants (revoked);