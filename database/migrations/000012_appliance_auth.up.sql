CREATE TABLE app_users (
    id UUID PRIMARY KEY,
    username VARCHAR(128) NOT NULL,
    email VARCHAR(320),
    password_hash TEXT NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'administrator',
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT app_users_role_check CHECK (role = 'administrator')
);

CREATE UNIQUE INDEX app_users_username_ci_idx ON app_users (LOWER(username));
CREATE UNIQUE INDEX app_users_email_ci_idx ON app_users (LOWER(email)) WHERE email IS NOT NULL;

CREATE TABLE app_auth_provider (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    provider VARCHAR(32) NOT NULL DEFAULT 'local',
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO app_auth_provider (singleton, provider) VALUES (TRUE, 'local') ON CONFLICT (singleton) DO NOTHING;

CREATE TABLE app_auth_allowlist (
    id UUID PRIMARY KEY,
    provider VARCHAR(32) NOT NULL,
    subject_type VARCHAR(16) NOT NULL CHECK (subject_type IN ('email', 'group')),
    subject_value VARCHAR(320) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(provider, subject_type, subject_value)
);

CREATE TABLE app_web_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES app_users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX app_web_sessions_expiry_idx ON app_web_sessions (expires_at);

GRANT SELECT, INSERT, UPDATE, DELETE ON app_users, app_auth_provider, app_auth_allowlist, app_web_sessions TO ogsd_api;
