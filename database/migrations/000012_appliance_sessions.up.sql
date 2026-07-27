CREATE TABLE appliance_sessions (
    token_hash BYTEA PRIMARY KEY,
    username VARCHAR(64) NOT NULL,
    csrf_hash BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX appliance_sessions_expires_at_idx ON appliance_sessions (expires_at);
CREATE INDEX appliance_sessions_username_idx ON appliance_sessions (username);

GRANT SELECT, INSERT, UPDATE, DELETE ON appliance_sessions TO ogsd_api;
