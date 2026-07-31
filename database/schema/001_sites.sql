CREATE TABLE sites (
                       id UUID PRIMARY KEY,
                       name VARCHAR(100) NOT NULL,
                       location VARCHAR(255),
                       upstream_site_ids TEXT[] NOT NULL DEFAULT '{}',
                       hub_device_ids TEXT[] NOT NULL DEFAULT '{}',
                       created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sites_upstream_site_ids ON sites USING GIN (upstream_site_ids);