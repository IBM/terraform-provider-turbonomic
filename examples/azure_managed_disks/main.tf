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
