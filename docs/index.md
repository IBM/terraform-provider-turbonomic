---
layout: ""
page_title: "Provider: IBM Turbonomic"
description: |-
  The Turbonomic provider supplies data resources to interact with the Turbonomic API.
---

# IBM Turbonomic provider

The Turbonomic provider supplies data resources to interact with the Turbonomic API.

The provider sets Turbonomic as the source of truth for scaling decisions about cloud
entities that are deployed through Terraform. The data source of the provider returns
the tier size based on Turbonomic scaling action recommendations, or the current size
if no scaling action exists. We recommend you include a default tier size for cases
where the entity is not deployed yet.

## Examples

The following examples demonstrate the Terraform code to configure the provider and use it in different cloud environments.

### Configure the provider credentials

Before you can use the provider, you must configure it with the hostname and credentials.

```terraform
terraform {
  required_providers {
    turbonomic = {
      source  = "IBM/turbonomic"
      version = "1.0.0"
    }
  }
}

provider "turbonomic" {
  username   = var.username
  password   = var.password
  hostname   = var.hostname
  skipverify = var.skipverify
}
```

### Example data source configuration

The following example shows the Turbonomic data source struct that you can use in different cloud environments.

```terraform
data "turbonomic_cloud_entity_recommendation" "example" {
  entity_name = "exampleVirtualMachine"
  entity_type = "VirtualMachine"
}
```

### AWS example

The _instance_type_ is configured to use the turbonomic_cloud_entity_recommendation data source unless
null is returned.  If this happens, a default of "t3.nano" is used.

```terraform
provider "aws" {
  region = "us-east-1"
}

data "turbonomic_cloud_entity_recommendation" "example" {
  entity_name = "exampleVirtualMachine"
  entity_type = "VirtualMachine"
}

resource "aws_instance" "terraform-demo-ec2" {
  ami = "ami-079db87dc4c10ac91"
  instance_type = (
    data.turbonomic_cloud_entity_recommendation.example.new_instance_type != null
    ? data.turbonomic_cloud_entity_recommendation.example.new_instance_type
    : "t3.nano"
  )
  tags = {
    Name = "terraform-demo-ec2"
  }
}
```

### Azure example

The `size` is configured to use the `turbonomic_cloud_entity_recommendation` data source unless null is
returned, in which case it uses `Standard_B1s` by default.

```terraform
provider "azurerm" {
  features {}
}

data "turbonomic_cloud_entity_recommendation" "example" {
  entity_name = "exampleVirtualMachine"
  entity_type = "VirtualMachine"
}

resource "azurerm_linux_virtual_machine" "exampleVirtualMachine" {
  name                = "exampleVirtualMachine"
  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
  size = (
    data.turbonomic_cloud_entity_recommendation.example.new_instance_type != null
    ? data.turbonomic_cloud_entity_recommendation.example.new_instance_type
    : "Standard_B1s"
  )
  admin_username        = "azureuser"
  network_interface_ids = [azurerm_network_interface.nic.id]

  admin_ssh_key {
    username   = "azureuser"
    public_key = file("~/.ssh/id_rsa.pub")
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "UbuntuServer"
    sku       = "18.04-LTS"
    version   = "latest"
  }
}
```

### GCP example

The `machine_type` is set to use the `turbonomic_cloud_entity_recommendation` data source unless null is returned, in which case it uses `e2-micro` by default.

```terraform
provider "google" {
  project = "example-turbonomic-terraform"
  region  = "us-central1"
}

resource "google_compute_instance" "exampleVirtualMachine" {
  name = "exampleVirtualMachine"
  machine_type = (
    data.turbonomic_cloud_entity_recommendation.example.new_instance_type != null
    ? data.turbonomic_cloud_entity_recommendation.example.new_instance_type
    : "e2-micro"
  )
  zone = "us-central1-a"

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-11"
    }
  }

  network_interface {
    network = "default"
    access_config {}
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `hostname` (String) Hostname or IP Address of Turbonomic Instance
- `password` (String, Sensitive) Password for the username to access the Turbonomic Instance
- `skipverify` (Boolean) Boolean on whether to verify the SSL/TLS certificate for the hostname
- `username` (String) Username to access the Turbonomic Instance
