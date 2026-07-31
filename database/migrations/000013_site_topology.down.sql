DROP INDEX IF EXISTS idx_sites_upstream_site_ids;

ALTER TABLE sites
  DROP COLUMN IF EXISTS hub_device_ids,
  DROP COLUMN IF EXISTS upstream_site_ids;
