ALTER TABLE sites
  ADD COLUMN IF NOT EXISTS upstream_site_ids TEXT[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS hub_device_ids TEXT[] NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_sites_upstream_site_ids
  ON sites USING GIN (upstream_site_ids);
