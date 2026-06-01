CREATE TABLE sites (
                       id UUID PRIMARY KEY,
                       name VARCHAR(100) NOT NULL,
                       location VARCHAR(255),
                       created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);