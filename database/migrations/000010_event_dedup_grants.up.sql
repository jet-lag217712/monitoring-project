-- V2 event_id deduplication ledger and role grants for new tables.

CREATE TABLE ingested_events (
    event_id UUID PRIMARY KEY,
    event_type VARCHAR(40) NOT NULL,
    site_id VARCHAR(100) NOT NULL,
    collector_id VARCHAR(100),
    device_id VARCHAR(100),
    observed_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ingested_events_observed ON ingested_events(observed_at DESC);
CREATE INDEX idx_ingested_events_site_type ON ingested_events(site_id, event_type);

-- Ingestion grants for v2 tables.
GRANT SELECT, INSERT, UPDATE ON
  device_temperature_components,
  device_power_components,
  device_health_current,
  collectors,
  collector_status_current
TO ogsd_ingestion;

GRANT SELECT, INSERT ON
  device_temperature_readings,
  device_power_readings,
  device_health_history,
  collector_heartbeat_history,
  ingested_events
TO ogsd_ingestion;

GRANT USAGE, SELECT ON SEQUENCE
  device_temperature_readings_id_seq,
  device_power_readings_id_seq,
  device_health_history_id_seq,
  collector_heartbeat_history_id_seq
TO ogsd_ingestion;

-- Devices/interfaces already have INSERT/UPDATE; ensure SELECT covers new columns via table grants.
GRANT SELECT ON
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
TO ogsd_api;
