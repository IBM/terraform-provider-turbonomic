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
