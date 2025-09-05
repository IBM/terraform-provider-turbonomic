data "turbonomic_azurerm_managed_disk" "example" {
  entity_name                  = "<entity_name>"
  default_storage_account_type = "<default_storage_account_type>"
  default_disk_iops_read_write = var.default_disk_iops_read_write
  default_disk_size_gb         = var.default_disk_size_gb
  default_disk_mbps_read_write = var.default_disk_mbps_read_write
}
