DROP INDEX IF EXISTS idx_alerts_created;
DROP INDEX IF EXISTS idx_alerts_device;
DROP INDEX IF EXISTS idx_interface_samples_interface_time;
DROP INDEX IF EXISTS idx_metric_samples_metric_time;
DROP INDEX IF EXISTS idx_metric_samples_device_time;
DROP INDEX IF EXISTS idx_interfaces_device;
DROP INDEX IF EXISTS idx_devices_site;

DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS interface_samples;
DROP TABLE IF EXISTS metric_samples;
DROP TABLE IF EXISTS interfaces;
DROP TABLE IF EXISTS metric_types;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS sites;
