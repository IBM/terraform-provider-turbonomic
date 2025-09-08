data "turbonomic_google_compute_disk" "example" {
  entity_name                    = "<entity_name>"
  default_type                   = "<default_type"
  default_provisioned_iops       = var.default_provisioned_iops
  default_provisioned_throughput = var.default_provisioned_throughput
  default_size                   = var.default_size
}
