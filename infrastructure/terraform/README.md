# Azure PostgreSQL (Terraform)

Provisions Azure Database for PostgreSQL **Flexible Server** in a dedicated resource group (separate failure domain from app hosts).

## Layout

```
infrastructure/terraform/
├── modules/postgresql/     # Reusable Flexible Server + database + firewall rules
├── environments/dev/       # Public access optional (local migrate)
└── environments/prod/      # Private VNet access + stronger SKU/backups
```

## Prerequisites

- Terraform >= 1.5
- Azure CLI authenticated (`az login`) — run only with explicit approval in agent sessions
- Unique `server_name` globally

## Dev apply (manual)

```bash
cd infrastructure/terraform/environments/dev
cp terraform.tfvars.example terraform.tfvars
# edit server_name, password, firewall IP

terraform init
terraform plan
terraform apply
```

Then migrate:

```bash
export DATABASE_URL='postgres://ogsd_admin:<password>@<fqdn>:5432/ogsd?sslmode=require'
./infrastructure/script/migrate.sh up
OGSD_INGESTION_PASSWORD='...' OGSD_API_PASSWORD='...' \
  ./infrastructure/script/bootstrap-db-roles.sh
```

## Prod notes

- Default: `public_network_access_enabled = false`
- Requires `delegated_subnet_id` and `private_dns_zone_id`
- Geo-redundant backups enabled by default
- Store admin password in Key Vault / CI secrets; never commit `terraform.tfvars`
