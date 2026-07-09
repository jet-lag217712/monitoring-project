terraform {
  required_version = ">= 1.5.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }
}

provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "db" {
  name     = var.resource_group_name
  location = var.location
  tags     = var.tags
}

module "postgresql" {
  source = "../../modules/postgresql"

  name                         = var.server_name
  resource_group_name          = azurerm_resource_group.db.name
  location                     = azurerm_resource_group.db.location
  postgresql_version           = var.postgresql_version
  sku_name                     = var.sku_name
  storage_mb                   = var.storage_mb
  administrator_login          = var.administrator_login
  administrator_password       = var.administrator_password
  database_name                = var.database_name
  backup_retention_days        = var.backup_retention_days
  geo_redundant_backup_enabled = var.geo_redundant_backup_enabled

  public_network_access_enabled = var.public_network_access_enabled
  delegated_subnet_id           = var.delegated_subnet_id
  private_dns_zone_id           = var.private_dns_zone_id
  firewall_rules                = var.firewall_rules

  tags = var.tags
}
