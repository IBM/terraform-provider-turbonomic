---
layout: ""
page_title: "Azure Examples"
description: |-
 This guide focuses on different azure resources examples.
---

This guide focuses on using Turbonomic data sources with [Azure](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs) resources, enabling dynamic resource configuration based on Turbonomic recommendations.

## Azure virtual machines example

The `size` is configured to use the `turbonomic_cloud_entity_recommendation` data source unless null is
returned, in which case it uses `Standard_B1s` by default.

```terraform
provider "azurerm" {
  features {}
}

data "turbonomic_cloud_entity_recommendation" "example" {
  entity_name  = "exampleVirtualMachine"
  entity_type  = "VirtualMachine"
  default_size = "Standard_B1s"
}

resource "azurerm_linux_virtual_machine" "exampleVirtualMachine" {
  name                  = "exampleVirtualMachine"
  resource_group_name   = azurerm_resource_group.rg.name
  location              = azurerm_resource_group.rg.location
  size                  = data.turbonomic_cloud_entity_recommendation.example.new_instance_type
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

The `storage_account_type` is set to use the `turbonomic_azurerm_managed_disk` data source unless null is returned, in which case it uses `Standard_LRS` by default.
```terraform
data "turbonomic_azurerm_managed_disk" "example" {
  entity_name                  = "example-azuredata-disk"
  default_storage_account_type = "Standard_LRS"
}

resource "azurerm_managed_disk" "example_disk" {
  name                 = "example-azuredata-disk"
  location             = "East US"
  resource_group_name  = "AppInfra_Integrations"
  storage_account_type = data.turbonomic_azurerm_managed_disk.example.new_storage_account_type
  create_option        = "Empty"
  disk_size_gb         = 32
  tags                 = provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider
}
```
