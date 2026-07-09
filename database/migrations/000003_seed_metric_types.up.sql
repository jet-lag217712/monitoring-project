-- Seed core metric types for MVP collector metrics.
INSERT INTO metric_types (id, name, unit, description)
VALUES (
  'a0000000-0000-4000-8000-000000000001',
  'uptime_seconds',
  'seconds',
  'Device sysUpTime converted to seconds'
)
ON CONFLICT (name) DO NOTHING;
