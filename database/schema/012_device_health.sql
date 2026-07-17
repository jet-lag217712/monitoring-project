-- Device health current state and history (migration 000008).

CREATE TABLE device_health_current (
    device_id UUID PRIMARY KEY REFERENCES devices(id),
    site_id UUID NOT NULL REFERENCES sites(id),
    state VARCHAR(20) NOT NULL,
    reason VARCHAR(40) NOT NULL,
    transition VARCHAR(20) NOT NULL,
    previous_state VARCHAR(20),
    failure_count INTEGER NOT NULL DEFAULT 0,
    failure_threshold INTEGER NOT NULL DEFAULT 2,
    temperature_c DOUBLE PRECISION,
    temperature_warning_c DOUBLE PRECISION,
    temperature_policy_revision VARCHAR(128),
    upstream_device_ids TEXT[] NOT NULL DEFAULT '{}',
    unavailable_upstream_device_ids TEXT[] NOT NULL DEFAULT '{}',
    root_cause_device_ids TEXT[] NOT NULL DEFAULT '{}',
    observed_at TIMESTAMPTZ NOT NULL,
    event_id UUID NOT NULL,
    config_revision VARCHAR(128),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE device_health_history (
    id BIGSERIAL PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES devices(id),
    site_id UUID NOT NULL REFERENCES sites(id),
    state VARCHAR(20) NOT NULL,
    reason VARCHAR(40) NOT NULL,
    transition VARCHAR(20) NOT NULL,
    previous_state VARCHAR(20),
    failure_count INTEGER NOT NULL DEFAULT 0,
    failure_threshold INTEGER NOT NULL DEFAULT 2,
    temperature_c DOUBLE PRECISION,
    temperature_warning_c DOUBLE PRECISION,
    temperature_policy_revision VARCHAR(128),
    upstream_device_ids TEXT[] NOT NULL DEFAULT '{}',
    unavailable_upstream_device_ids TEXT[] NOT NULL DEFAULT '{}',
    root_cause_device_ids TEXT[] NOT NULL DEFAULT '{}',
    observed_at TIMESTAMPTZ NOT NULL,
    event_id UUID NOT NULL UNIQUE,
    config_revision VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_device_health_history_device_time
  ON device_health_history(device_id, observed_at DESC);
CREATE INDEX idx_device_health_current_site
  ON device_health_current(site_id);
