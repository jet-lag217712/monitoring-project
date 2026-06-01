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