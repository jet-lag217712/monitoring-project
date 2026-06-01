CREATE TABLE alerts (
                        id UUID PRIMARY KEY,

                        device_id UUID REFERENCES devices(id),

                        interface_id UUID REFERENCES interfaces(id),

                        severity VARCHAR(20) NOT NULL,

                        alert_type VARCHAR(100) NOT NULL,

                        message TEXT NOT NULL,

                        acknowledged BOOLEAN NOT NULL DEFAULT FALSE,

                        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

                        cleared_at TIMESTAMPTZ
);