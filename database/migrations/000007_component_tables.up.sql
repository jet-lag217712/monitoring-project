-- Temperature and power component inventory + readings.

CREATE TABLE device_temperature_components (
    id UUID PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES devices(id),
    component_id VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    component_index VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (device_id, component_id)
);

CREATE TABLE device_temperature_readings (
    id BIGSERIAL PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES devices(id),
    component_id VARCHAR(128) NOT NULL,
    value DOUBLE PRECISION,
    unit VARCHAR(25) NOT NULL,
    status VARCHAR(50) NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    event_id UUID NOT NULL,
    UNIQUE (device_id, component_id, observed_at)
);

CREATE TABLE device_power_components (
    id UUID PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES devices(id),
    component_id VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    component_index VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (device_id, component_id)
);

CREATE TABLE device_power_readings (
    id BIGSERIAL PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES devices(id),
    component_id VARCHAR(128) NOT NULL,
    value DOUBLE PRECISION,
    unit VARCHAR(25) NOT NULL,
    status VARCHAR(50) NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    event_id UUID NOT NULL,
    UNIQUE (device_id, component_id, observed_at)
);

CREATE INDEX idx_temp_components_device ON device_temperature_components(device_id);
CREATE INDEX idx_temp_readings_device_time ON device_temperature_readings(device_id, observed_at DESC);
CREATE INDEX idx_power_components_device ON device_power_components(device_id);
CREATE INDEX idx_power_readings_device_time ON device_power_readings(device_id, observed_at DESC);
