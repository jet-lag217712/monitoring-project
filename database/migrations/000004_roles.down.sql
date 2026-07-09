REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM ogsd_api, ogsd_ingestion, ogsd_admin;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM ogsd_ingestion, ogsd_admin;
REVOKE USAGE ON SCHEMA public FROM ogsd_api, ogsd_ingestion, ogsd_admin;

DO $$
BEGIN
  EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM ogsd_api, ogsd_ingestion, ogsd_admin', current_database());
EXCEPTION
  WHEN undefined_object THEN NULL;
END
$$;

DROP ROLE IF EXISTS ogsd_api;
DROP ROLE IF EXISTS ogsd_ingestion;
-- Do not DROP ogsd_admin: on Azure it may be the server administrator.
