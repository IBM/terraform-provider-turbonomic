provider "azurerm" {
  features {}
}

variable "sql_admin_password" {
  description = "SQL Server administrator password"
  type        = string
  sensitive   = true
}

data "turbonomic_azurerm_mssql_database" "example" {
  entity_name      = "<entity_name>"
  default_sku_name = "<default_sku_name>"
}

# Resource group
resource "azurerm_resource_group" "example" {
  name     = "example-resources"
  location = "East US"
}

# SQL Server
resource "azurerm_mssql_server" "example" {
  name                         = "example-sqlserver"
  resource_group_name          = azurerm_resource_group.example.name
  location                     = azurerm_resource_group.example.location
  version                      = "12.0"
  administrator_login          = "sqladminuser"
  administrator_login_password = var.sql_admin_password
}

# SQL Database
resource "azurerm_mssql_database" "example" {
  name        = "example-db"
  server_id   = azurerm_mssql_server.example.id
  sku_name    = data.turbonomic_azurerm_mssql_database.example.new_sku_name
  max_size_gb = 10
  collation   = "SQL_Latin1_General_CP1_CI_AS"
}
