-- Additive device/interface enrichment for SNMP Collector v2 (migration 000006).
-- Full devices/interfaces definitions remain in 002_devices.sql / 004_interfaces.sql.
-- vendor, model, and snmp_version already exist on devices from migration 1.

-- devices additive columns:
--   serial, sys_object_id, sys_name, sys_descr,
--   profile_name, capabilities, collector_id, config_revision, last_observed_at

-- interfaces additive columns:
--   if_alias, if_type, last_observed_at
