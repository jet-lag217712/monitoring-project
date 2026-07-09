variable "name" {
  description = "PostgreSQL Flexible Server name (globally unique)."
  type        = string
}

variable "resource_group_name" {
  description = "Resource group for the Flexible Server."
  type        = string
}

variable "location" {
  description = "Azure region."
  type        = string
}

variable "postgresql_version" {
  description = "PostgreSQL major version."
  type        = string
  default     = "16"
}

variable "sku_name" {
  description = "Flexible Server SKU (e.g. B_Standard_B1ms, GP_Standard_D2s_v3)."
  type        = string
  default     = "B_Standard_B1ms"
}

variable "storage_mb" {
  description = "Storage size in MB."
  type        = number
  default     = 32768
}

variable "administrator_login" {
  description = "Server administrator login (use ogsd_admin to match migration roles)."
  type        = string
  default     = "ogsd_admin"
}

variable "administrator_password" {
  description = "Server administrator password. Prefer TF_VAR_administrator_password."
  type        = string
  sensitive   = true
}

variable "database_name" {
  description = "Application database name."
  type        = string
  default     = "ogsd"
}

variable "backup_retention_days" {
  description = "Backup retention in days (7–35)."
  type        = number
  default     = 7
}

variable "geo_redundant_backup_enabled" {
  description = "Enable geo-redundant backups."
  type        = bool
  default     = false
}

variable "zone" {
  description = "Availability zone (optional)."
  type        = string
  default     = null
}

variable "public_network_access_enabled" {
  description = "Allow public network access (dev convenience). Prefer false in prod with private access."
  type        = bool
  default     = false
}

variable "delegated_subnet_id" {
  description = "Subnet ID for private Flexible Server (VNet integration). Null when using public access."
  type        = string
  default     = null
}

variable "private_dns_zone_id" {
  description = "Private DNS zone ID for privatelink.postgres.database.azure.com. Required with delegated_subnet_id."
  type        = string
  default     = null
}

variable "firewall_rules" {
  description = "Public firewall rules (ignored when public_network_access_enabled is false)."
  type = list(object({
    name             = string
    start_ip_address = string
    end_ip_address   = string
  }))
  default = []
}

variable "tags" {
  description = "Resource tags."
  type        = map(string)
  default     = {}
}
