REVOKE DELETE ON
  metric_samples,
  interface_samples,
  device_temperature_readings,
  device_power_readings,
  device_health_history,
  collector_heartbeat_history,
  ingested_events,
  alerts
FROM ogsd_ingestion;
