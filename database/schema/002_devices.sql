CREATE TABLE devices (
                         id UUID PRIMARY KEY,

                         site_id UUID NOT NULL REFERENCES sites(id),

                         hostname VARCHAR(255) NOT NULL,

                         ip_address INET NOT NULL,

                         vendor VARCHAR(50) NOT NULL,

                         model VARCHAR(100) NOT NULL,

                         snmp_version VARCHAR(20) NOT NULL,

                         status VARCHAR(20) NOT NULL DEFAULT 'unknown',

                         last_seen TIMESTAMPTZ,

                         created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);