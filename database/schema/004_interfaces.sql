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