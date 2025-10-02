provider "aws" {
  region = "us-east-1"
}

data "turbonomic_aws_db_instance" "rdsExample" {
  entity_name               = "<entity_name>"
  default_instance_class    = "<default_instance_class>"
  default_storage_type      = "<default_storage_type>"
  default_iops              = var.iops
  default_allocated_storage = var.allocated_storage
}

resource "aws_db_instance" "default" {
  identifier           = "<entity_name>"
  allocated_storage    = data.turbonomic_aws_db_instance.rdsExample.new_allocated_storage
  db_name              = "mydb"
  engine               = "mysql"
  engine_version       = "8.0"
  instance_class       = data.turbonomic_aws_db_instance.rdsExample.default_instance_class
  storage_type         = data.turbonomic_aws_db_instance.rdsExample.default_storage_type
  iops                 = data.turbonomic_aws_db_instance.rdsExample.new_iops
  username             = "dbuser"
  password             = "dbpassword"
  parameter_group_name = "default.mysql8.0"
  skip_final_snapshot  = true
  tags                 = provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider
}
