CREATE TABLE metric_samples (
                                id BIGSERIAL PRIMARY KEY,

                                device_id UUID NOT NULL REFERENCES devices(id),

                                metric_type_id UUID NOT NULL REFERENCES metric_types(id),

                                value DOUBLE PRECISION NOT NULL,

                                collected_at TIMESTAMPTZ NOT NULL
);