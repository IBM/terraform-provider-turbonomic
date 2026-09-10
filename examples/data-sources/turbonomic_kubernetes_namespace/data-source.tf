data "turbonomic_kubernetes_namespace" "example" {
  cluster = "<cluster_name>"
  name    = "<namespace>"

  # Optional fallback values used when Turbonomic has no recommendation
  default_cpu_limit_quota   = "2000"
  default_cpu_request_quota = "1000"
  default_mem_limit_quota   = "4096"
  default_mem_request_quota = "2048"
}
