provider "aws" {
  region = "us-east-1"
}

data "turbonomic_aws_ebs_volume" "example" {
  entity_name  = "exampleEBSVolumeName"
  default_type = "gp2"
}

resource "aws_ebs_volume" "example-ebs-instance" {
  availability_zone = "us-east-1a"
  size              = 100
  type              = data.turbonomic_aws_ebs_volume.example.new_type
  iops              = 3000
  throughput        = 125
  tags = merge(
    {
      Name = "exampleEBSVolumeName"
    },
    provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider
  )
}
