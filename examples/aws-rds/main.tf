provider "aws" {
  region = "us-east-1"
}

data "turbonomic_aws_db_instance" "rdsExample" {
  entity_name            = "exampleDBinstance"
  default_instance_class = "db.t3.small"
  default_storage_type   = "gp2"
}


resource "aws_db_instance" "default" {
  identifier           = "exampleDBinstance"
  allocated_storage    = 10
  db_name              = "mydb"
  engine               = "mysql"
  engine_version       = "8.0"
  instance_class       = data.turbonomic_aws_db_instance.rdsExample.default_instance_class
  storage_type         = data.turbonomic_aws_db_instance.rdsExample.default_storage_type
  username             = "dbuser"
  password             = "dbpassword"
  parameter_group_name = "default.mysql8.0"
  skip_final_snapshot  = true
  tags                 = provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider
}
