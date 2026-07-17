-- Collector inventory, current status, and heartbeat history.

CREATE TABLE collectors (
    id UUID PRIMARY KEY,
    site_id UUID NOT NULL REFERENCES sites(id),
    collector_id VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (site_id, collector_id)
);

CREATE TABLE collector_status_current (
    collector_uuid UUID PRIMARY KEY REFERENCES collectors(id),
    site_id UUID NOT NULL REFERENCES sites(id),
    collector_id VARCHAR(100) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    version VARCHAR(128) NOT NULL,
    git_commit VARCHAR(128) NOT NULL,
    build_time VARCHAR(128) NOT NULL,
    uptime_seconds DOUBLE PRECISION NOT NULL,
    sqlite_queue_depth BIGINT NOT NULL,
    memory_usage_bytes BIGINT NOT NULL,
    goroutine_count INTEGER NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    event_id UUID NOT NULL,
    config_revision VARCHAR(128),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE collector_heartbeat_history (
    id BIGSERIAL PRIMARY KEY,
    collector_uuid UUID NOT NULL REFERENCES collectors(id),
    site_id UUID NOT NULL REFERENCES sites(id),
    collector_id VARCHAR(100) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    version VARCHAR(128) NOT NULL,
    git_commit VARCHAR(128) NOT NULL,
    build_time VARCHAR(128) NOT NULL,
    uptime_seconds DOUBLE PRECISION NOT NULL,
    sqlite_queue_depth BIGINT NOT NULL,
    memory_usage_bytes BIGINT NOT NULL,
    goroutine_count INTEGER NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    event_id UUID NOT NULL UNIQUE,
    config_revision VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_collector_heartbeat_collector_time
  ON collector_heartbeat_history(collector_uuid, observed_at DESC);
CREATE INDEX idx_collector_status_site
  ON collector_status_current(site_id);
