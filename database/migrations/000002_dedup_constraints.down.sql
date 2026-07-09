ALTER TABLE interface_samples
  DROP CONSTRAINT IF EXISTS uq_interface_sample_idempotency;

ALTER TABLE metric_samples
  DROP CONSTRAINT IF EXISTS uq_metric_sample_idempotency;
