---
page_title: "Enabling HCP Continuous Validation"
subcategory: ""
description: |-
  The following demonstrates configuring your Terraform Repository and HCP to use Continuous Validation.
---

# HashiCorp Cloud Platform Continuous Validation
The Continuous Validation feature of HashiCorp Cloud Platform (HCP) Terraform can be used, along with the Turbonomic Terraform Provider setup, to periodically assess and identify adherence of Terraform resource configurations with Turbonomic's recommendation.  The periodic assessment runs every 24 hours and can also be run manually from the specific page of the Continuous Validation in HCP.

The steps to be followed are as follows
Use the following sections to setup Continuous Validation:
- [Enable health assessment](#enable-health-check---continuous-validation-in-hcp) in the workspace you wish to use Continuous Validation
- [Add check blocks](#add-check-blocks-in-terraform-configuration-file-as-shown-below) to your Terraform configuration code in those workspaces
- [Verify](#after-applying-the-configuration---check-assertion-result-will-be-available-in-continuous-validation-link-of-hcp) that the Continuous Validation is working as expected in HCP under Health Continuous Validation section

## Enable health check

To enable health check in your HCP Terraform workspace, use the following steps:

1. Sign in to HCP Terraform or Terraform Enterprise.
1. Navigate to the workspace you want to enable health assessments on.
1. Verify that your workspace meets the necessary requirements. For example, Terraform version, execution mode.
1. Click **Settings** in the workspace.
1. Click **Health**.
1. Select **Enable** under Health Assessments.
1. Click **Save settings**.

Once enabled, HCP Terraform periodically runs health assessments to ensure your infrastructure matches the defined configuration.

![HCP Health Settings](https://github.com/IBM/terraform-provider-turbonomic/blob/main/imgs/hcp-healthsettings-enable.png?raw=true)

Ref: https://developer.hashicorp.com/terraform/cloud-docs/workspaces/health

## Add check blocks

The following shows the configuration setup to be added to perform the assertion check of an AWS EC2 instance with Turbonomic recommendation:

1. Add the provider configuration and define the resources.
1. Set datasource for getting the Turbonomic recommendations.
1. Add `turbonomic_recommendation_check` assertion:

```terraform
terraform {
  required_providers {
    turbonomic = {
      source  = "IBM/turbonomic"
      version = "1.2.0"
    }
  }
}

provider "turbonomic" {
  hostname   = var.hostname
  username   = var.username
  password   = var.password
  skipverify = var.skipverify
}

//provider block for AWS
provider "aws" {
  region = "us-east-1"
}

//data block which retrieves recommendations from Turbonomic for an entity
data "turbonomic_cloud_entity_recommendation" "example" {
  entity_name = "terraform-example-ec2"
  entity_type = "VirtualMachine"
}

resource "aws_instance" "terraform-instance-1" {
  ami = "ami-079db87dc4c10ac91"
  instance_type = "t3.nano"

  tags = {
    Name = "terraform-example-ec2"
  }
}

//Gives a warning incase of failure and display on the continuous health check
check "turbonomic_recommendation_check"{

  assert {
    condition =  aws_instance.terraform-instance-1.instance_type == coalesce(data.turbonomic_cloud_entity_recommendation.example.new_instance_type,aws_instance.terraform-instance-1.instance_type)
    error_message = "Must use the latest recommended instance type,${coalesce(data.turbonomic_cloud_entity_recommendation.example.new_instance_type,aws_instance.terraform-instance-1.instance_type)}"
  }

}
```

Note: Ensure the Turbonomic provider is added as per the [provider configuration](https://registry.terraform.io/providers/IBM/turbonomic/latest/docs#configure-the-provider-credentials).  For more details regarding the location of your Terraform code, please contact your Terraform code owners.

## Verify
After applying the configuration, check assertion result is available in Continuous Validation link of HCP.

To view and verify the assertions,follow these steps:

1. Log into your HCP Terraform or Terraform Enterprise account.
1. Go to the specific workspace where you have enabled health assessments.
1. Click on the **Health** tab within the workspace settings.
1. Within the **Health** section, click on **Continuous Validation** to view the results of the latest health assessments.
1. Click on **Start health assessment** to do a manual check.

![HCP Health Settings](https://github.com/IBM/terraform-provider-turbonomic/blob/main/imgs/continous_validation_hcp.png?raw=true)
