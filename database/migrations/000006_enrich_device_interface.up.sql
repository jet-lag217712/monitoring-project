-- Additive device identity / profile / fingerprint columns (v1 columns retained).
-- Note: vendor, model, and snmp_version already exist on devices from migration 1.
ALTER TABLE devices
  ADD COLUMN IF NOT EXISTS serial VARCHAR(255),
  ADD COLUMN IF NOT EXISTS sys_object_id VARCHAR(255),
  ADD COLUMN IF NOT EXISTS sys_name VARCHAR(255),
  ADD COLUMN IF NOT EXISTS sys_descr TEXT,
  ADD COLUMN IF NOT EXISTS profile_name VARCHAR(50),
  ADD COLUMN IF NOT EXISTS capabilities TEXT[],
  ADD COLUMN IF NOT EXISTS collector_id VARCHAR(100),
  ADD COLUMN IF NOT EXISTS config_revision VARCHAR(128),
  ADD COLUMN IF NOT EXISTS last_observed_at TIMESTAMPTZ;

-- Additive interface metadata columns (v1 columns retained).
-- Note: name, description, admin_status, oper_status, speed_bps already exist.
ALTER TABLE interfaces
  ADD COLUMN IF NOT EXISTS if_alias VARCHAR(255),
  ADD COLUMN IF NOT EXISTS if_type VARCHAR(100),
  ADD COLUMN IF NOT EXISTS last_observed_at TIMESTAMPTZ;
