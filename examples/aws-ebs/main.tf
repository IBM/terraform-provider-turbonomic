provider "aws" {
  region = "us-east-1"
}

data "turbonomic_aws_ebs_volume" "example" {
  entity_name        = "<entity_name>"
  default_type       = "<default_type>"
  default_iops       = var.iops
  default_size       = var.size
  default_throughput = var.throughput
}

resource "aws_ebs_volume" "example-ebs-instance" {
  availability_zone = "us-east-1a"
  size              = data.turbonomic_aws_ebs_volume.example.new_size
  type              = data.turbonomic_aws_ebs_volume.example.new_type
  iops              = data.turbonomic_aws_ebs_volume.example.new_iops
  throughput        = data.turbonomic_aws_ebs_volume.example.new_throughput
  tags = merge(
    {
      Name = "<entity_name>"
    },
    provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider
  )
}
