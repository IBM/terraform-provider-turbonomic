---
page_title: "Turbonomic provider in modifiable custom Terraform modules"
subcategory: ""
description: |-
  The following describes detail on how to add Turbonomic provider in custom modules.
---

# Turbonomic provider in modifiable custom Terraform modules

[Terraform modules](https://developer.hashicorp.com/terraform/language/modules) are a clean and scalable way to package and reuse resource configurations. Any combination of resources and other constructs can be factored into a module. It helps to reuse configuration and standardize deployments.

## Adding Turbonomic provider to a modifiable custom module

Assume that you have a [custom module](https://developer.hashicorp.com/terraform/language/modules/develop) named `ec2_turbonomic_module` for creating AWS EC2 instances. Turbonomic provider data block can be configured in the custom module as shown in the following example:

Here,`turbonomic_aws_instance` data block fetches the `new_instance_type` recommendation from Turbonomic. The `entity_name`, which is passed as input to the module for creating the EC2 resource has to be re-used for Turbonomic data block as well. The `instance_type` is configured to use the `turbonomic_aws_instance` data block unless null is returned. If null is returned, then the `default_instance_type` defined in data block of Turbonomic is used.

```diff

resource "aws_instance" "ec2_instance" {
  ami           = var.ami
<span style='color:green'>+ instance_type = data.turbonomic_aws_instance.ec2_instance_recommendation.new_instance_type </span>
  tags          = merge({
    Name        = var.name                            //name of the EC2 instance
  },var.tags
  )
}
<span style='color:green'>
+ data "turbonomic_aws_instance" "ec2_instance_recommendation" {
+    entity_name  = var.name                         //name of the EC2 instance
+    default_instance_type  = var.default_instance_type       //default instance type to be used
+}
</span>
```

If Turbonomic recommendation for instance types is needed by external callers, then configure the output block within the custom module definition using the following example:

```hcl
output "turbonomic_module_out" {
  value = data.turbonomic_aws_instance.ec2_instance_recommendation
}
```

## Invoking the module

Terraform modules can be invoked from another modules or default root module. Terraform supports dynamic blocks through `for_each` and `count` meta-argument to the modules.

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

To create EC2 instance for each configuration, `for_each` is used inside the module invocation of `ec2_turbonomic_module`.

```diff
module "ec2_turbonomic_module"{
  source                = "./Modules/ec2_turbonomic_module"      //module location
  <mark>for_each              = local.instance_ami_types</mark> 		       //reading values from config
  ami                   = each.value.ami					     //reading ami from config  <span style='color:green'>
+ name                  = each.value.name                        //reading name from config
+ default_instance_type = each.value.instance_type	           //inside the module,Turbonomic reads it as the default instance
+ tags                  = provider::turbonomic::get_tag()        //tag the resource as optimised as Turbonomic provider </span>
}</span>
```

### Reading Turbonomic recommendation

Turbonomic recommendations can be accessed within the invoking module as seen from the following example:

```hcl

#To read Turbonomic recommendations for each EC2 instance from module
output "turbonomic_output" {
    value = {
        for key,ec2_instance in module.ec2_turbonomic_module: key => ec2_instance.turbonomic_module_out
    }
}

```

## Case 2 - same configuration for multiple instances (count based)

If the requirement is to create multiple instances of the same configuration, `count` based approach can be used.

```diff
module "ec2_turbonomic_module"{
  source                = "./Modules/ec2_turbonomic_module"             //module location
  <mark>count                 = var.ec2_instances_count</mark> 		   		      //reading values from config
  ami                   = var.ami									   //reading ami from config  <span style='color:green'>
+ name                  = "exampleVirtualMachine-trb-${count.index}"    //indexed based name
+ default_instance_type = var.instance_type						     //inside the module, Turbonomic reads it as the default instance
+ tags                  = provider::turbonomic::get_tag()               //tag the resource as optimised as Turbonomic provider </span>
}
```

### Reading Turbonomic recommendation

Turbonomic recommendations can be accessed within the invoking module as seen from the following example:


```hcl

#To get read recommendations of Turbonomic for all resource
output "turbonomic_output" {
    value = module.ec2_turbonomic_module[*].turbonomic_module_out
}

```
