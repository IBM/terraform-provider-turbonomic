data "turbonomic_azurerm_managed_disk" "example" {
  entity_name                  = "exampleAzureDisk"
  default_storage_account_type = "Standard_B1s"
}
