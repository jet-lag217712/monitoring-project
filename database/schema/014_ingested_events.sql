-- V2 event_id deduplication ledger (migration 000010).

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
