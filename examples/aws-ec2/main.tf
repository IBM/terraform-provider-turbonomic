provider "aws" {
  region = "us-east-1"
}

data "turbonomic_cloud_entity_recommendation" "example" {
  entity_name  = "exampleVirtualMachine"
  entity_type  = "VirtualMachine"
  default_size = "t3.nano"
}

resource "aws_instance" "terraform-demo-ec2" {
  ami           = "ami-079db87dc4c10ac91"
  instance_type = data.turbonomic_cloud_entity_recommendation.example.new_instance_type
  tags = {
    Name = "terraform-demo-ec2"
  }
}
