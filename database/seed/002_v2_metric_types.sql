-- Seed v2 device metric types (also applied via migration 000005).
INSERT INTO metric_types (id, name, unit, description)
VALUES
  (
    'a0000000-0000-4000-8000-000000000002',
    'cpu_utilization_pct',
    'percent',
    'CPU utilization percentage (0-100)'
  ),
  (
    'a0000000-0000-4000-8000-000000000003',
    'memory_utilization_pct',
    'percent',
    'Memory utilization percentage (0-100)'
  ),
  (
    'a0000000-0000-4000-8000-000000000004',
    'primary_temperature_c',
    'celsius',
    'Primary device temperature in degrees Celsius'
  ),
  (
    'a0000000-0000-4000-8000-000000000005',
    'power_state',
    'state',
    'Power-supply component state reading'
  ),
  (
    'a0000000-0000-4000-8000-000000000006',
    'power_watts',
    'watts',
    'Power-supply power reading in watts'
  ),
  (
    'a0000000-0000-4000-8000-000000000007',
    'power_volts',
    'volts',
    'Power-supply voltage reading in volts'
  ),
  (
    'a0000000-0000-4000-8000-000000000008',
    'power_amps',
    'amps',
    'Power-supply current reading in amps'
  )
ON CONFLICT (name) DO NOTHING;
