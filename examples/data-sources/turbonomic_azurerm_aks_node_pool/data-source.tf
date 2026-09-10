data "turbonomic_azurerm_aks_node_pool" "example" {
  cluster_name       = "<cluster_name>"
  name               = "<node_pool_name>"
  default_node_count = 3
}
