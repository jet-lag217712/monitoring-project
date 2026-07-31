ALTER TABLE device_health_history
    DROP COLUMN IF EXISTS alerts_enabled;

ALTER TABLE device_health_current
    DROP COLUMN IF EXISTS alerts_enabled;
