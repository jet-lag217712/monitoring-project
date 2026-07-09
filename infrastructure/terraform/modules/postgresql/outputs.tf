output "server_id" {
  description = "Flexible Server resource ID."
  value       = azurerm_postgresql_flexible_server.this.id
}

output "server_fqdn" {
  description = "Fully qualified domain name of the Flexible Server."
  value       = azurerm_postgresql_flexible_server.this.fqdn
}

output "database_name" {
  description = "Application database name."
  value       = azurerm_postgresql_flexible_server_database.ogsd.name
}

output "administrator_login" {
  description = "Administrator login name."
  value       = azurerm_postgresql_flexible_server.this.administrator_login
}

output "connection_hint" {
  description = "Non-secret connection hint for migrations (password from secrets store)."
  value       = "postgres://${azurerm_postgresql_flexible_server.this.administrator_login}:<password>@${azurerm_postgresql_flexible_server.this.fqdn}:5432/${azurerm_postgresql_flexible_server_database.ogsd.name}?sslmode=require"
}
