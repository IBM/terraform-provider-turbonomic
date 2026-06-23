provider "azurerm" {
  features {}
}

# Query existing Azure VM (if it exists)
data "azurerm_linux_virtual_machine" "existing" {
  name                = "my-linux-vm"
  resource_group_name = "my-resource-group"
}

# Query Turbonomic for recommendations
data "turbonomic_azurerm_linux_virtual_machine" "example" {
  entity_name = "my-linux-vm"
  vendor_id   = data.azurerm_linux_virtual_machine.existing.id
  # default_size is intentionally omitted to enable fallback pattern
}

# Create or update the VM with fallback logic
# The coalesce() function creates a priority chain:
# 1. Use Turbonomic recommendation if available
# 2. Use current Azure VM size if Turbonomic is unavailable
# 3. Use default "Standard_B1s" for new VMs
resource "azurerm_linux_virtual_machine" "example" {
  name                = "my-linux-vm"
  resource_group_name = "my-resource-group"
  location            = "eastus"
  size = coalesce(
    data.turbonomic_azurerm_linux_virtual_machine.example.new_size, # Turbonomic recommendation
    try(data.azurerm_linux_virtual_machine.existing.size, null),    # Current Azure VM size
    "Standard_B1s"                                                  # Default for new VMs
  )

  admin_username = "adminuser"

  network_interface_ids = [
    azurerm_network_interface.example.id,
  ]

  admin_ssh_key {
    username   = "adminuser"
    public_key = file("~/.ssh/id_rsa.pub")
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts"
    version   = "latest"
  }

  tags = provider::turbonomic::get_tag() # Tag the resource as optimized by Turbonomic provider
}

# Output the selected VM size for verification
output "selected_vm_size" {
  description = "The VM size selected by the fallback pattern"
  value       = azurerm_linux_virtual_machine.example.size
}

output "turbonomic_recommendation" {
  description = "The VM size recommended by Turbonomic (if available)"
  value       = data.turbonomic_azurerm_linux_virtual_machine.example.new_size
}

output "current_vm_size" {
  description = "The current Azure VM size (if exists)"
  value       = try(data.azurerm_linux_virtual_machine.existing.size, "N/A - New VM")
}
