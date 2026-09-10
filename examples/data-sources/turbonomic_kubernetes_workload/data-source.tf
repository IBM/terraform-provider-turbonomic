data "turbonomic_kubernetes_workload" "example" {
  cluster   = "<cluster_name>"
  namespace = "<namespace>"
  name      = "<workload_name>"

  default_replicas = 1

  default_containers = [
    {
      name                   = "<container_name>"
      default_cpu_request    = "200m"
      default_cpu_limit      = "500m"
      default_memory_request = "256Mi"
      default_memory_limit   = "512Mi"
    }
  ]
}
