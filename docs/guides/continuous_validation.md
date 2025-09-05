---
page_title: "Enabling HCP Continuous validation"
subcategory: ""
description: |-
  The following demonstrates configuring your Terraform Repository and HCP to use Continuous validation.
---

# Enabling HCP Continuous validation

The Continuous validation feature of HashiCorp Cloud Platform (HCP) Terraform can be used, along with the Turbonomic Terraform Provider setup, to periodically assess and identify adherence of Terraform resource configurations with Turbonomic's recommendation.  The periodic assessment runs every 24 hours and can also be run manually from the specific page of the Continuous validation in HCP.

Use the following sections to setup Continuous validation:
- [Enable health assessment](#enable-health-check---continuous-validation-in-hcp) in the workspace you wish to use Continuous validation
- [Add check blocks](#add-check-blocks-in-terraform-configuration-file-as-shown-below) to your Terraform configuration code in those workspaces
- [Verify](#after-applying-the-configuration---check-assertion-result-will-be-available-in-continuous-validation-link-of-hcp) that the Continuous validation is working as expected in HCP under Health Continuous validation section

## Enabling health assessment

To enable health assessment in your HCP Terraform workspace, use the following steps:

1. Sign in to HCP Terraform or Terraform Enterprise.
1. Navigate to the workspace you want to enable health assessments on.
1. Verify that your workspace meets the necessary requirements. For example, Terraform version, execution mode.
1. Click **Settings** in the workspace.
1. Click **Health**.
1. Select **Enable** under Health Assessments.
1. Click **Save settings**.

Once enabled, HCP Terraform periodically runs health assessments to ensure your infrastructure matches the defined configuration.

![HCP Health Settings](https://github.com/IBM/terraform-provider-turbonomic/blob/main/imgs/hcp-healthsettings-enable.png?raw=true)

For more information, see [Health assessment](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/health).

## Adding check blocks

Use the following configuration setup to perform the assertion check of an AWS EC2 instance with Turbonomic recommendation:

1. Add the provider configuration and define the resources.
1. Set datasource for getting the Turbonomic recommendations.
1. Add `turbonomic_recommendation_check` assertion:

```terraform
terraform {
  required_providers {
    turbonomic = {
      source  = "IBM/turbonomic"
      version = "1.5.0"
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

//data block which retrieves recommendations from Turbonomic for AWS EC2 instance
data "turbonomic_aws_instance" "example" {
  entity_name = "terraform-example-ec2"
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
    condition =  aws_instance.terraform-instance-1.instance_type == coalesce(data.turbonomic_aws_instance.example.new_instance_type,aws_instance.terraform-instance-1.instance_type)
    error_message = "Must use the latest recommended instance type,${coalesce(data.turbonomic_aws_instance.example.new_instance_type,aws_instance.terraform-instance-1.instance_type)}"
  }

}
```

Note: Ensure the Turbonomic provider is added as per the [provider configuration](https://registry.terraform.io/providers/IBM/turbonomic/latest/docs#configure-the-provider-credentials).  For more details about the location of your Terraform code, contact your Terraform code owners.

## Verifying
After applying the configuration, check assertion result is available in Continuous validation link of HCP.

To view and verify the assertions, use the following steps:

1. Log into your HCP Terraform or Terraform Enterprise account.
1. Go to the specific workspace where you have enabled health assessments.
1. Click **Health** tab within the workspace settings.
1. Within the **Health** section, click **Continuous validation** to view the results of the latest health assessments.
1. Click **Start health assessment** to do a manual check.

![HCP Health Settings](https://github.com/IBM/terraform-provider-turbonomic/blob/main/imgs/continous_validation_hcp.png?raw=true)
