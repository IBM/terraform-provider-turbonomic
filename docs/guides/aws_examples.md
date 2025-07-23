---
layout: ""
page_title: "AWS Examples"
description: |-
 This guide focuses on different aws resources examples.
---

This guide focuses on using Turbonomic data sources with [AWS](https://registry.terraform.io/providers/hashicorp/aws/latest/docs) resources, enabling dynamic resource configuration based on Turbonomic recommendations.

## AWS EC2 example

The `instance_type` is configured to use the turbonomic_cloud_entity_recommendation data source unless
null is returned.  If null is returned, then the default size defined in data block of "t3.nano" is used.

```terraform
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

  tags = merge(
    {
      Name = "exampleVirtualMachine"
    },
    provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider
  )
}
```
## AWS RDS example

The AWS RDS resource is configured to use the `turbonomic_aws_db_instance` data source unless null is returned, in which case it defaults to `db.t3.small` for the
`default_instance_class` and `gp2` for the `default_storage_type`.

```terraform
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
```
## AWS EBS example

The AWS EBS resource is configured to use the `turbonomic_aws_ebs_volume` data source unless null is returned, in which case it defaults to `gp2` for the `default_type`.

```terraform
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
```
