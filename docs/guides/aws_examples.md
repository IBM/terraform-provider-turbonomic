---
layout: ""
page_title: "AWS examples"
description: |-
 This guide focuses on different aws resources examples.
---

# AWS examples

This guide focuses on using Turbonomic data sources with [AWS](https://registry.terraform.io/providers/hashicorp/aws/latest/docs) resources, enabling dynamic resource configuration based on Turbonomic recommendations.

## AWS EC2 example

The AWS EC2 resource is configured to use the `turbonomic_aws_instance` data source unless null is returned, in which case it uses `<default_instance_type>` by default.

```terraform
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
```

## AWS RDS example

The AWS RDS resource is configured to use the `turbonomic_aws_db_instance` data source unless null is returned, in which case it defaults to `<default_instance_class>` for the `default_instance_class` and `<default_storage_type>` for the `default_storage_type`.

```terraform
provider "aws" {
  region = "us-east-1"
}

data "turbonomic_aws_db_instance" "rdsExample" {
  entity_name            = "<entity_name>"
  default_instance_class = "<default_instance_class>"
  default_storage_type   = "<default_storage_type>"
}

resource "aws_db_instance" "default" {
  identifier           = "<entity_name>"
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

The AWS EBS resource is configured to use the `turbonomic_aws_ebs_volume` data source unless null is returned, in which case it uses `<default_type>` by default.

```terraform
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
```
