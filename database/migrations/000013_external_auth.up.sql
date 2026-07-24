CREATE TABLE IF NOT EXISTS app_external_identities (
    provider TEXT NOT NULL,
    subject TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (provider, subject),
    UNIQUE (user_id, provider)
);

GRANT SELECT, INSERT, UPDATE, DELETE ON app_external_identities TO ogsd_api;
