output "resource_group_name" {
  value = azurerm_resource_group.db.name
}

output "server_fqdn" {
  value = module.postgresql.server_fqdn
}

output "database_name" {
  value = module.postgresql.database_name
}

output "administrator_login" {
  value = module.postgresql.administrator_login
}

output "connection_hint" {
  value = module.postgresql.connection_hint
}
