variable "resource_group_name" {
  description = "Resource group for the prod PostgreSQL failure domain."
  type        = string
  default     = "ogsd-pg-prod-rg"
}

variable "location" {
  description = "Azure region."
  type        = string
  default     = "eastus"
}

variable "server_name" {
  description = "Globally unique Flexible Server name."
  type        = string
}

variable "postgresql_version" {
  type    = string
  default = "16"
}

variable "sku_name" {
  description = "Prod SKU — general purpose recommended."
  type        = string
  default     = "GP_Standard_D2s_v3"
}

variable "storage_mb" {
  type    = number
  default = 131072
}

variable "administrator_login" {
  type    = string
  default = "ogsd_admin"
}

variable "administrator_password" {
  description = "Admin password. Set via TF_VAR_administrator_password or a secret store."
  type        = string
  sensitive   = true
}

variable "database_name" {
  type    = string
  default = "ogsd"
}

variable "backup_retention_days" {
  type    = number
  default = 14
}

variable "geo_redundant_backup_enabled" {
  type    = bool
  default = true
}

variable "public_network_access_enabled" {
  description = "Prod default: private access only."
  type        = bool
  default     = false
}

variable "delegated_subnet_id" {
  description = "Required for private Flexible Server in prod."
  type        = string
}

variable "private_dns_zone_id" {
  description = "Required with delegated_subnet_id."
  type        = string
}

variable "firewall_rules" {
  type = list(object({
    name             = string
    start_ip_address = string
    end_ip_address   = string
  }))
  default = []
}

variable "tags" {
  type = map(string)
  default = {
    project     = "ogsd"
    environment = "prod"
    managed_by  = "terraform"
  }
}
