-- Initial schema (from database/schema/001–008).
CREATE TABLE sites (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    location VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE devices (
    id UUID PRIMARY KEY,
    site_id UUID NOT NULL REFERENCES sites(id),
    hostname VARCHAR(255) NOT NULL,
    ip_address INET NOT NULL,
    vendor VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,
    snmp_version VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    last_seen TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE metric_types (
    id UUID PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    unit VARCHAR(25),
    description TEXT
);

CREATE TABLE interfaces (
    id UUID PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES devices(id),
    if_index INTEGER NOT NULL,
    name VARCHAR(255),
    description TEXT,
    admin_status VARCHAR(20),
    oper_status VARCHAR(20),
    speed_bps BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, if_index)
);

CREATE TABLE metric_samples (
    id BIGSERIAL PRIMARY KEY,
    device_id UUID NOT NULL REFERENCES devices(id),
    metric_type_id UUID NOT NULL REFERENCES metric_types(id),
    value DOUBLE PRECISION NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE interface_samples (
    id BIGSERIAL PRIMARY KEY,
    interface_id UUID NOT NULL REFERENCES interfaces(id),
    in_octets BIGINT,
    out_octets BIGINT,
    in_errors BIGINT,
    out_errors BIGINT,
    in_discards BIGINT,
    out_discards BIGINT,
    collected_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE alerts (
    id UUID PRIMARY KEY,
    device_id UUID REFERENCES devices(id),
    interface_id UUID REFERENCES interfaces(id),
    severity VARCHAR(20) NOT NULL,
    alert_type VARCHAR(100) NOT NULL,
    message TEXT NOT NULL,
    acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cleared_at TIMESTAMPTZ
);

CREATE INDEX idx_devices_site ON devices(site_id);
CREATE INDEX idx_interfaces_device ON interfaces(device_id);
CREATE INDEX idx_metric_samples_device_time ON metric_samples(device_id, collected_at DESC);
CREATE INDEX idx_metric_samples_metric_time ON metric_samples(metric_type_id, collected_at DESC);
CREATE INDEX idx_interface_samples_interface_time ON interface_samples(interface_id, collected_at DESC);
CREATE INDEX idx_alerts_device ON alerts(device_id);
CREATE INDEX idx_alerts_created ON alerts(created_at DESC);
