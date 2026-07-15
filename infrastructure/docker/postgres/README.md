# Local / reference PostgreSQL

OGSD uses stock `postgres:16-alpine` for local development. There is no custom image.

**Local stack:** [`deployments/development/`](../../../deployments/development/) — compose service `postgres`, migrations via [`../../script/migrate.sh`](../../script/migrate.sh).

**Cloud:** Azure Database for PostgreSQL Flexible Server — [`../../terraform/modules/postgresql/`](../../terraform/modules/postgresql/).
