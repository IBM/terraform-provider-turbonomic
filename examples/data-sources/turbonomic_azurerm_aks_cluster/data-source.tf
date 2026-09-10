data "turbonomic_azurerm_aks_cluster" "example" {
  name          = "<cluster_name>"
  default_state = "RUNNING"
}
