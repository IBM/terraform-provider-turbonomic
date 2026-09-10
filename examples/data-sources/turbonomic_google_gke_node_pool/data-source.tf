data "turbonomic_google_gke_node_pool" "example" {
  cluster_name       = "<cluster_name>"
  node_pool_name     = "<node_pool_name>"
  default_node_count = 3
}
