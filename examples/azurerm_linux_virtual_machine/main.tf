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
