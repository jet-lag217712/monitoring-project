CREATE TABLE metric_types (
                              id UUID PRIMARY KEY,

                              name VARCHAR(100) UNIQUE NOT NULL,

                              unit VARCHAR(25),

                              description TEXT
);