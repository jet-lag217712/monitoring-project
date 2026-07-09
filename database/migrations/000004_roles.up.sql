-- Application roles. Passwords are set by infrastructure/script/bootstrap-db-roles.sh
-- (never commit production passwords). On Azure, ogsd_admin is typically the Flexible
-- Server administrator created by Terraform; create it here only if missing (local).

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'ogsd_admin') THEN
    CREATE ROLE ogsd_admin WITH LOGIN NOSUPERUSER CREATEDB CREATEROLE NOINHERIT;
  END IF;

  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'ogsd_ingestion') THEN
    CREATE ROLE ogsd_ingestion WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;

  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'ogsd_api') THEN
    CREATE ROLE ogsd_api WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT;
  END IF;
END
$$;

DO $$
BEGIN
  EXECUTE format('GRANT CONNECT ON DATABASE %I TO ogsd_admin, ogsd_ingestion, ogsd_api', current_database());
END
$$;

GRANT USAGE ON SCHEMA public TO ogsd_admin, ogsd_ingestion, ogsd_api;

-- Admin: full DML + DDL on monitoring objects (migrations / ops).
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ogsd_admin;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ogsd_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ogsd_admin;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO ogsd_admin;
-- When migrations run as ogsd_admin (Azure), keep future objects readable by API.
ALTER DEFAULT PRIVILEGES FOR ROLE ogsd_admin IN SCHEMA public GRANT SELECT ON TABLES TO ogsd_api;
ALTER DEFAULT PRIVILEGES FOR ROLE ogsd_admin IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE ON TABLES TO ogsd_ingestion;
ALTER DEFAULT PRIVILEGES FOR ROLE ogsd_admin IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO ogsd_ingestion;

-- Ingestion: auto-discovery upserts + sample inserts.
-- SELECT on sample tables is required for ON CONFLICT idempotency checks.
GRANT SELECT ON sites, devices, interfaces, metric_types, metric_samples, interface_samples, alerts TO ogsd_ingestion;
GRANT INSERT, UPDATE ON sites, devices, interfaces TO ogsd_ingestion;
GRANT INSERT ON metric_samples, interface_samples TO ogsd_ingestion;
GRANT USAGE, SELECT ON SEQUENCE metric_samples_id_seq, interface_samples_id_seq TO ogsd_ingestion;

-- API: read-only.
GRANT SELECT ON sites, devices, interfaces, metric_types, metric_samples, interface_samples, alerts TO ogsd_api;

ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO ogsd_api;
