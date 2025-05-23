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
if no scaling action exists.

## Examples

The following examples demonstrate the Terraform code to and use the provider in different cloud environments.

### Configuring the provider credentials

Before you can use the provider, you must configure it with credentials.  You can use either username and password or OAuth 2.0.

#### Example using username/password credentials

```terraform
terraform {
  required_providers {
    turbonomic = {
      source  = "IBM/turbonomic"
      version = "1.1.0"
    }
  }
}

provider "turbonomic" {
  hostname   = var.hostname
  username   = var.username
  password   = var.password
  skipverify = var.skipverify
}
```

#### Example using OAuth 2.0 credentials

In order to authenticate to Turbonomic's API using OAuth 2.0, you first need to create an OAuth client.
For more information, see [Creating and authenticating an OAuth 2.0 client](https://www.ibm.com/docs/en/tarm/8.15.0?topic=cookbook-authenticating-oauth-20-clients-api#cookbook_administration_oauth_authentication__title__4)
to create the client.  Note, you must create the OAuth 2.0 client using `client_secret_basic` as the `clientAuthenticationMethods`. The output from the preceeding documentation will result in the following parameters:
- clientId
- clientSecret
- role

```terraform
terraform {
  required_providers {
    turbonomic = {
      source  = "IBM/turbonomic"
      version = "1.1.0"
    }
  }
}

provider "turbonomic" {
  hostname      = var.hostname
  client_id     = var.client_id
  client_secret = var.client_secret
  role          = var.role
  skipverify    = var.skipverify
}
```

**Note:** Valid roles are ADMINISTRATOR, SITE_ADMIN, AUTOMATOR, DEPLOYER, ADVISOR, OBSERVER, OPERATIONAL_OBSERVER, SHARED_ADVISOR, SHARED_OBSERVER and REPORT_EDITOR.

### Example data source configuration

The following example shows the Turbonomic data source struct that you can use in different cloud environments.

```terraform
data "turbonomic_cloud_entity_recommendation" "example" {
  entity_name  = "exampleVirtualMachine"
  entity_type  = "VirtualMachine"
  default_size = "defaultSize"
}
```

### AWS example

The _instance_type_ is configured to use the turbonomic_cloud_entity_recommendation data source unless
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
  entity_name  = "exampleVirtualMachine"
  entity_type  = "VirtualMachine"
  default_size = "Standard_B1s"
}

resource "azurerm_linux_virtual_machine" "exampleVirtualMachine" {
  name                  = "exampleVirtualMachine"
  resource_group_name   = azurerm_resource_group.rg.name
  location              = azurerm_resource_group.rg.location
  size                  = data.turbonomic_cloud_entity_recommendation.example.new_instance_type
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

data "turbonomic_cloud_entity_recommendation" "example" {
  entity_name  = "exampleVirtualMachine"
  entity_type  = "VirtualMachine"
  default_size = "n1-standard-1"
}

resource "google_compute_instance" "exampleVirtualMachine" {
  name         = "exampleVirtualMachine"
  machine_type = data.turbonomic_cloud_entity_recommendation.example.new_instance_type
  zone         = "us-central1-a"

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

- `client_id` (String) the OAuth 2.0 client ID that can be used to access the Turbonomic instance; use TURBO_CLIENT_ID to set with an enviornment variable
- `client_secret` (String, Sensitive) the OAuth 2.0 client secret that can be used to access the Turbonomic instance; use TURBO_CLIENT_SECRET to set with an enviornment variable
- `hostname` (String) hostname or IP Address of Turbonomic Instance; use TURBO_HOSTNAME to set with an enviornment variable
- `password` (String, Sensitive) password for the username to access the Turbonomic Instance; use TURBO_PASSWORD to set with an enviornment variable
- `role` (String) the OAuth 2.0 role that can be used to access the Turbonomic instance; use TURBO_ROLE to set with an enviornment variable
- `skipverify` (Boolean) boolean on whether to verify the SSL or TLS certificate for the hostname
- `username` (String) username to access the Turbonomic Instance; use TURBO_USERNAME to set with an enviornment variable
