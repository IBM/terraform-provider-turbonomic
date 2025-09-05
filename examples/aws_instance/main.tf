provider "aws" {
  region = "us-east-1"
}

data "turbonomic_aws_instance" "example" {
  entity_name           = "<entity_name>"
  default_instance_type = "<default_instance_type>"
}

resource "aws_instance" "terraform-demo-ec2" {
  ami           = "ami-079db87dc4c10ac91"
  instance_type = data.turbonomic_aws_instance.example.new_instance_type

  tags = merge(
    {
      Name = "<entity_name>"
    },
    provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider
  )
}
