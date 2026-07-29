# Local PostgreSQL

Equate uses stock `postgres:16-alpine` inside the local appliance. There is no
custom database image.

The Compose service is `postgres`; migrations are applied from
[`../../script/migrate.sh`](../../script/migrate.sh). PostgreSQL is attached to
the private appliance network and is never published as a customer-facing
endpoint.
