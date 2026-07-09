-- Idempotency constraints for ingestion (Phase 3 / Phase 4).
ALTER TABLE metric_samples
  ADD CONSTRAINT uq_metric_sample_idempotency
  UNIQUE (device_id, metric_type_id, collected_at);

ALTER TABLE interface_samples
  ADD CONSTRAINT uq_interface_sample_idempotency
  UNIQUE (interface_id, collected_at);
