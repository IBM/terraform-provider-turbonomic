data "turbonomic_kubernetes_volume" "example" {
  cluster = "<cluster_name>"
  name    = "<pvc_name>"

  # Optional fallback size used when Turbonomic has no SCALE recommendation
  default_size_gib = 10
}
