CREATE INDEX idx_devices_site
    ON devices(site_id);

CREATE INDEX idx_interfaces_device
    ON interfaces(device_id);

CREATE INDEX idx_metric_samples_device_time
    ON metric_samples(device_id, collected_at DESC);

CREATE INDEX idx_metric_samples_metric_time
    ON metric_samples(metric_type_id, collected_at DESC);

CREATE INDEX idx_interface_samples_interface_time
    ON interface_samples(interface_id, collected_at DESC);

CREATE INDEX idx_alerts_device
    ON alerts(device_id);

CREATE INDEX idx_alerts_created
    ON alerts(created_at DESC);