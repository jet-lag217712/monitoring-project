variable "resource_group_name" {
  description = "Resource group for the dev PostgreSQL failure domain."
  type        = string
  default     = "ogsd-pg-dev-rg"
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
  type    = string
  default = "B_Standard_B1ms"
}

variable "storage_mb" {
  type    = number
  default = 32768
}

variable "administrator_login" {
  type    = string
  default = "ogsd_admin"
}

variable "administrator_password" {
  description = "Admin password. Set via TF_VAR_administrator_password or terraform.tfvars (gitignored)."
  type        = string
  sensitive   = true
}

variable "database_name" {
  type    = string
  default = "ogsd"
}

variable "backup_retention_days" {
  type    = number
  default = 7
}

variable "geo_redundant_backup_enabled" {
  type    = bool
  default = false
}

variable "public_network_access_enabled" {
  description = "Dev default: public access with firewall rules for local migrate."
  type        = bool
  default     = true
}

variable "delegated_subnet_id" {
  type    = string
  default = null
}

variable "private_dns_zone_id" {
  type    = string
  default = null
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
    environment = "dev"
    managed_by  = "terraform"
  }
}
