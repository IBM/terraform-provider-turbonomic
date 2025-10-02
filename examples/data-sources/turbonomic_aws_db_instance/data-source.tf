data "turbonomic_aws_db_instance" "testRDS2" {
  entity_name               = "<entity_name>"
  default_instance_class    = "<default_instance_class>"
  default_storage_type      = "<default_storage_type>"
  default_iops              = var.iops
  default_allocated_storage = var.allocated_storage
}
