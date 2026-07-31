-- Persist Administratively Ignored (alerts_enabled=false) from health events.
ALTER TABLE device_health_current
    ADD COLUMN IF NOT EXISTS alerts_enabled BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE device_health_history
    ADD COLUMN IF NOT EXISTS alerts_enabled BOOLEAN NOT NULL DEFAULT TRUE;
