data "turbonomic_kubernetes_pod" "example" {
  cluster   = "<cluster_name>"
  namespace = "<namespace>"
  name      = "<pod_name>"
}
