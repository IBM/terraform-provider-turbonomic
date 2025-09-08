data "turbonomic_aws_ebs_volume" "example" {
  entity_name        = "<entity_name>"
  default_type       = "<default_type>"
  default_iops       = var.iops
  default_size       = var.size
  default_throughput = var.throughput
}
