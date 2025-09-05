---
layout: ""
page_title: "Azure examples"
description: |-
 This guide focuses on different azure resources examples.
---

# Azure examples

This guide focuses on using Turbonomic data sources with [Azure](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs) resources, enabling dynamic resource configuration based on Turbonomic recommendations.

## Azure virtual machines example

The `size` is configured to use the `turbonomic_azurerm_linux_virtual_machine` data source unless null is returned, in which case it uses `<default_size>` by default.

```terraform
provider "azurerm" {
  features {}
}

data "turbonomic_azurerm_linux_virtual_machine" "example" {
  entity_name  = "<entity_name>"
  default_size = "<default_size>"
}

resource "azurerm_linux_virtual_machine" "exampleVirtualMachine" {
  name                  = "<entity_name>"
  resource_group_name   = azurerm_resource_group.rg.name
  location              = azurerm_resource_group.rg.location
  size                  = data.turbonomic_azurerm_linux_virtual_machine.example.new_size
  admin_username        = "azureuser"
  network_interface_ids = [azurerm_network_interface.nic.id]
  tags                  = provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider

  admin_ssh_key {
    username   = "azureuser"
    public_key = file("~/.ssh/id_rsa.pub")
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "18.04-LTS"
    version   = "latest"
  }
}
```

## Azure managed disks example

The `storage_account_type` is set to use the `turbonomic_azurerm_managed_disk` data source unless null is returned, in which case it uses `<default_storage_account_type>` by default.

```terraform
data "turbonomic_azurerm_managed_disk" "example" {
  entity_name                  = "<entity_name>"
  default_storage_account_type = "<default_storage_account_type>"
  default_disk_iops_read_write = var.default_disk_iops_read_write
  default_disk_size_gb         = var.default_disk_size_gb
  default_disk_mbps_read_write = var.default_disk_mbps_read_write
}

resource "azurerm_managed_disk" "example_disk" {
  name                 = "<entity_name>"
  location             = "East US"
  resource_group_name  = "AppInfra_Integrations"
  storage_account_type = data.turbonomic_azurerm_managed_disk.example.new_storage_account_type
  create_option        = "Empty"
  disk_size_gb         = data.turbonomic_azurerm_managed_disk.example.new_disk_size_gb
  disk_mbps_read_write = data.turbonomic_azurerm_managed_disk.example.new_disk_mbps_read_write
  disk_iops_read_write = data.turbonomic_azurerm_managed_disk.example.new_disk_iops_read_write
  tags                 = provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider
}
```

## Azure mssql database example

The `sku_name` is set to use the `turbonomic_azurerm_mssql_database` data source unless null is returned, in which case it uses `<default_sku_name>` by default.

```terraform
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
```
