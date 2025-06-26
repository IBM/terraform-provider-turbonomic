data "turbonomic_aws_db_instance" "testRDS2" {
  entity_name            = "exampleDBinstance"
  default_instance_class = "db.t3.small"
  default_storage_type   = "gp2"
}
