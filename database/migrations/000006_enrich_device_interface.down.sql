ALTER TABLE interfaces
  DROP COLUMN IF EXISTS last_observed_at,
  DROP COLUMN IF EXISTS if_type,
  DROP COLUMN IF EXISTS if_alias;

ALTER TABLE devices
  DROP COLUMN IF EXISTS last_observed_at,
  DROP COLUMN IF EXISTS config_revision,
  DROP COLUMN IF EXISTS collector_id,
  DROP COLUMN IF EXISTS capabilities,
  DROP COLUMN IF EXISTS profile_name,
  DROP COLUMN IF EXISTS sys_descr,
  DROP COLUMN IF EXISTS sys_name,
  DROP COLUMN IF EXISTS sys_object_id,
  DROP COLUMN IF EXISTS serial;
