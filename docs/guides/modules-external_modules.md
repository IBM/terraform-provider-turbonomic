---
page_title: "Turbonomic provider with external Terraform modules"
subcategory: ""
description: |-
  The following describes detail on how to add Turbonomic provider with external modules.
---

# Turbonomic provider with external Terraform modules

[Terraform modules](https://developer.hashicorp.com/terraform/language/modules) are a clean and scalable way to package and reuse resource configurations. Any combination of resources and other constructs can be factored into a module. It helps to reuse configuration and standardize deployments. Terraform modules can be invoked from another modules or default root module. Terraform supports dynamic blocks through `for_each` and `count` meta-argument to the modules.

Use the following examples to include Turbonomic provider while using an external or unmodifiable module, such as `terraform-aws-modules`.

## Case 1 - different configuration per instance (for_each based)

In this example, the configuration for each EC2 instance is different and is stored as a map object `instance_ami_types`. Hence, `for_each` based approach is used while invoking the module.

```hcl
locals {
    instance_ami_types = {
       "instance_ami_map1" = {
            name          = "example-1",
            instance_type = "t2.nano",
            ami           = "ami-079db87dc4c10ac91",
        },
        "instance_ami_map2" = {
            name          = "example-2",
            instance_type = "t3a.nano",
            ami           = "ami-079db87dc4c10ac91",
        },
    }
}
```

To create EC2 instance for each configuration, `for_each` is used inside the module invocation of `ec2_turbonomic_module` as well as inside the data block of Turbonomic to correlate the entity name.

The `instance_type` is configured to use the `turbonomic_cloud_entity_recommendation` data block unless null is returned. If null is returned, then the `default_size` defined in data block of Turbonomic is used. `each.key` is used to bind the reference of the entity from Turbonomic data block.


```diff
module "ec2_turbonomic_module"{
  source                = "terraform-aws-modules/ec2-instance/aws"        //mention the source from terraform registry
  version               = "4.3.0"                                         //mention the version from TF registry
  <mark>for_each              = local.instance_ami_types</mark>		 			     //reading values from config
  ami                   = each.value.ami								  //reading ami from config  <span style='color:green'>
+ name                  = each.value.name                                 //reading name from config
+ instance_type         = data.turbonomic_cloud_entity_recommendation.ec2_recommendation[each.key].new_instance_type		//reading
Turbonomic recommendation
+ tags                  = provider::turbonomic::get_tag()                  //tag the resource as optimised as Turbonomic provider </span>
}

#Turbonomic recommendation data block <span style='color:green'>
+ data "turbonomic_cloud_entity_recommendation" "ec2_recommendation" {
+  <mark>for_each     = local.instance_ami_types</mark>
+  entity_name  = each.value.name
+  entity_type  = "VirtualMachine"
+  default_size = each.value.instance_type                                //reading instance_type from configuration
+ }</span>
```

### Reading Turbonomic recommendation

Turbonomic recommendations can be accessed within the invoking module as seen from the following example:

```hcl

#To read Turbonomic recommendations for each EC2 instance from data block
output "turbonomic_output" {
    value = {
        for key,recommendation in data.turbonomic_cloud_entity_recommendation.ec2_recommendation: key => recommendation
    }
}

```

## Case 2 - same configuration for multiple instances (count based)

If the requirement is to create multiple instances of the same configuration, `count` based approach can be adopted. `count` is used inside the module invocation of `ec2_turbonomic_module` as well as inside the data block of Turbonomic to correlate the entity name.

The `instance_type` is configured to use the `turbonomic_cloud_entity_recommendation` data block unless null is returned. If null is returned, then the `default_size` defined in data block of Turbonomic is used. `count.index` is used to bind the reference of the entity from Turbonomic data block.

```diff
module "ec2_turbonomic_module"{
  source        = "terraform-aws-modules/ec2-instance/aws"         //mention the source from terraform registry
  version       = "4.3.0"                                          //mention the version from TF registry
  <mark>count         = var.ec2_instances_count</mark> 						 //reading values from config
  ami           = var.ami																			    //reading ami from config<span style='color:green'>
+ name          = "exampleVirtualMachine-trb-${count.index}"      //indexed based name
+ instance_type = data.turbonomic_cloud_entity_recommendation.ec2_recommendation[count.index].new_instance_type		//reading Turbonomic recommendation
+ tags          = provider::turbonomic::get_tag()                 //tag the resource as optimised as Turbonomic provider </span>
}

#Turbonomic recommendation data block <span style='color:green'>
+ data "turbonomic_cloud_entity_recommendation" "ec2_recommendation" {
+  <mark>count        = var.ec2_instances_count</mark>
+  entity_name  = "exampleVirtualMachine-trb-${count.index}"
+  entity_type  = "VirtualMachine"
+  default_size = var.instance_type                                //reading instance_type from configuration
+ }</span>
```

### Reading Turbonomic recommendation

Turbonomic recommendations can be accessed within the invoking module as seen from the following example:

```hcl

#To get read recommendations of Turbonomic for all resource
output "turbonomic_output" {
    value = data.turbonomic_cloud_entity_recommendation.ec2_recommendation[*].new_instance_type
}

```
