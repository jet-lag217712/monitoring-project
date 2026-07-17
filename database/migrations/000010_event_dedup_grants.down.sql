REVOKE SELECT ON
  device_temperature_components,
  device_temperature_readings,
  device_power_components,
  device_power_readings,
  device_health_current,
  device_health_history,
  collectors,
  collector_status_current,
  collector_heartbeat_history,
  ingested_events
FROM ogsd_api;

REVOKE USAGE, SELECT ON SEQUENCE
  device_temperature_readings_id_seq,
  device_power_readings_id_seq,
  device_health_history_id_seq,
  collector_heartbeat_history_id_seq
FROM ogsd_ingestion;

REVOKE SELECT, INSERT ON
  device_temperature_readings,
  device_power_readings,
  device_health_history,
  collector_heartbeat_history,
  ingested_events
FROM ogsd_ingestion;

REVOKE SELECT, INSERT, UPDATE ON
  device_temperature_components,
  device_power_components,
  device_health_current,
  collectors,
  collector_status_current
FROM ogsd_ingestion;

DROP INDEX IF EXISTS idx_ingested_events_site_type;
DROP INDEX IF EXISTS idx_ingested_events_observed;
DROP TABLE IF EXISTS ingested_events;
