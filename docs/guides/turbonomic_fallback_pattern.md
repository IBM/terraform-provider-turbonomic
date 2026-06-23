---
layout: ""
page_title: "Fallback Pattern for Turbonomic Unavailability"
description: |-
 This guide demonstrates how to handle Turbonomic unavailability gracefully using Terraform's coalesce() function across AWS, Azure, and Google Cloud.
---

# Fallback Pattern for Turbonomic Unavailability

You can use a Terraform fallback pattern to handle scenarios where Turbonomic recommendations are temporarily unavailable or when you manage both new and existing cloud resources.

The pattern uses the Terraform `coalesce()` function to ensure that infrastructure deployments continue without unintended configuration changes.

## Overview

When using Turbonomic recommendations in Terraform configurations, the following scenarios can occur:

1. A new virtual machine is being deployed and no existing instance configuration exists.
2. An existing virtual machine has a Turbonomic recommendation available.
3. An existing virtual machine does not have a Turbonomic recommendation because Turbonomic is temporarily unavailable or returns no recommendation.

To handle these scenarios consistently, use the Terraform `coalesce()` function to create a fallback priority chain.

The fallback order is:
1. Turbonomic recommendation
2. Existing cloud provider instance type
3. Default instance type

This approach helps prevent unintended resize operations when Turbonomic data is unavailable.

## The Pattern

The key to this pattern is using `coalesce()` to create a priority chain:

```
Turbonomic recommendation → Cloud provider current instance type → Default fallback
```

### How It Works

1. **First priority**: Use Turbonomic's recommendation if available
2. **Second priority**: If Turbonomic is unavailable or returns null, use the existing cloud provider instance type
3. **Third priority**: If neither exists (new VM), use a sensible default

## Implementation

### Step 1: Configure the Turbonomic Data Source

Set `default_instance_type = null` (or omit it entirely) in the Turbonomic data source. This allows the data source to return `null` when Turbonomic is unavailable, enabling the fallback chain:

```terraform
data "turbonomic_aws_instance" "example" {
  entity_name = "<entity_name>"
  vendor_id   = "<vendor_id>"
  # default_instance_type is omitted (null) to enable fallback pattern
}
```

### Step 2: Query Existing Cloud Provider Resource (Optional)

If managing an existing VM, query its current configuration:

```terraform
data "aws_instance" "existing" {
  instance_id = "i-xxxxx"  # or use filters to find existing instance
}
```

### Step 3: Use coalesce() in Resource Configuration

Apply the fallback chain using `coalesce()`:

```terraform
resource "aws_instance" "terraform-demo-ec2" {
  ami           = "ami-079db87dc4c10ac91"
  instance_type = coalesce(
    data.turbonomic_aws_instance.example.new_instance_type,  # Turbonomic recommendation
    data.aws_instance.existing.instance_type,                 # Current AWS instance type
    "t2.nano"                                                 # Default for new VMs
  )

  tags = merge(
    {
      Name = "<entity_name>"
    },
    provider::turbonomic::get_tag()
  )
}
```

## Examples by Cloud Provider

The fallback pattern works consistently across all major cloud providers. Below are examples for each.

### AWS EC2 Instance Example

```terraform
provider "aws" {
  region = "us-east-1"
}

# Query existing AWS instance (if it exists)
data "aws_instance" "existing" {
  instance_id = "i-0021d61fa77f000d0" # Replace with your instance ID or use filters
}

# Query Turbonomic for recommendations
data "turbonomic_aws_instance" "example" {
  entity_name = "my-ec2-instance"
  vendor_id   = "i-0021d61fa77f000d0"
  # default_instance_type is intentionally omitted to enable fallback pattern
}

# Create or update the instance with fallback logic
# The coalesce() function creates a priority chain:
# 1. Use Turbonomic recommendation if available
# 2. Use current AWS instance type if Turbonomic is unavailable
# 3. Use default "t2.nano" for new VMs
resource "aws_instance" "terraform-demo-ec2" {
  ami = "ami-079db87dc4c10ac91"
  instance_type = coalesce(
    data.turbonomic_aws_instance.example.new_instance_type, # Turbonomic recommendation
    try(data.aws_instance.existing.instance_type, null),    # Current AWS instance type
    "t2.nano"                                               # Default for new VMs
  )

  tags = merge(
    {
      Name = "my-ec2-instance"
    },
    provider::turbonomic::get_tag() # Tag the resource as optimized by Turbonomic provider
  )
}

# Output the selected instance type for verification
output "selected_instance_type" {
  description = "The instance type selected by the fallback pattern"
  value       = aws_instance.terraform-demo-ec2.instance_type
}

output "turbonomic_recommendation" {
  description = "The instance type recommended by Turbonomic (if available)"
  value       = data.turbonomic_aws_instance.example.new_instance_type
}

output "current_instance_type" {
  description = "The current AWS instance type (if exists)"
  value       = try(data.aws_instance.existing.instance_type, "N/A - New instance")
}
```

### Azure Virtual Machine Example

```terraform
provider "azurerm" {
  features {}
}

# Query existing Azure VM (if it exists)
data "azurerm_linux_virtual_machine" "existing" {
  name                = "my-linux-vm"
  resource_group_name = "my-resource-group"
}

# Query Turbonomic for recommendations
data "turbonomic_azurerm_linux_virtual_machine" "example" {
  entity_name = "my-linux-vm"
  vendor_id   = data.azurerm_linux_virtual_machine.existing.id
  # default_size is intentionally omitted to enable fallback pattern
}

# Create or update the VM with fallback logic
# The coalesce() function creates a priority chain:
# 1. Use Turbonomic recommendation if available
# 2. Use current Azure VM size if Turbonomic is unavailable
# 3. Use default "Standard_B1s" for new VMs
resource "azurerm_linux_virtual_machine" "example" {
  name                = "my-linux-vm"
  resource_group_name = "my-resource-group"
  location            = "eastus"
  size = coalesce(
    data.turbonomic_azurerm_linux_virtual_machine.example.new_size, # Turbonomic recommendation
    try(data.azurerm_linux_virtual_machine.existing.size, null),    # Current Azure VM size
    "Standard_B1s"                                                  # Default for new VMs
  )

  admin_username = "adminuser"

  network_interface_ids = [
    azurerm_network_interface.example.id,
  ]

  admin_ssh_key {
    username   = "adminuser"
    public_key = file("~/.ssh/id_rsa.pub")
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = "Standard_LRS"
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts"
    version   = "latest"
  }

  tags = provider::turbonomic::get_tag() # Tag the resource as optimized by Turbonomic provider
}

# Output the selected VM size for verification
output "selected_vm_size" {
  description = "The VM size selected by the fallback pattern"
  value       = azurerm_linux_virtual_machine.example.size
}

output "turbonomic_recommendation" {
  description = "The VM size recommended by Turbonomic (if available)"
  value       = data.turbonomic_azurerm_linux_virtual_machine.example.new_size
}

output "current_vm_size" {
  description = "The current Azure VM size (if exists)"
  value       = try(data.azurerm_linux_virtual_machine.existing.size, "N/A - New VM")
}
```

### Google Cloud Compute Instance Example

```terraform
provider "google" {
  project = "my-project"
  region  = "us-central1"
}

# Query existing GCP instance (if it exists)
# This data source will be used as a fallback when Turbonomic is unavailable
data "google_compute_instance" "existing" {
  name = "my-instance"
  zone = "us-central1-a"
}

# Query Turbonomic for recommendations
# Note: default_machine_type is omitted (null) to enable the fallback pattern
data "turbonomic_google_compute_instance" "example" {
  entity_name = "my-instance"
  vendor_id   = data.google_compute_instance.existing.id
  # default_machine_type is intentionally omitted to enable fallback pattern
}

# Create or update the instance with fallback logic
# The coalesce() function creates a priority chain:
# 1. Use Turbonomic recommendation if available
# 2. Use current GCP machine type if Turbonomic is unavailable
# 3. Use default "e2-micro" for new VMs
resource "google_compute_instance" "example" {
  name = "my-instance"
  machine_type = coalesce(
    data.turbonomic_google_compute_instance.example.new_machine_type, # Turbonomic recommendation
    try(data.google_compute_instance.existing.machine_type, null),    # Current GCP machine type
    "e2-micro"                                                        # Default for new VMs
  )
  zone = "us-central1-a"

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-11"
    }
  }

  network_interface {
    network = "default"
    access_config {
      // Ephemeral IP
    }
  }

  metadata = {
    for k, v in provider::turbonomic::get_tag() : k => v
  }
}

# Output the selected machine type for verification
output "selected_machine_type" {
  description = "The machine type selected by the fallback pattern"
  value       = google_compute_instance.example.machine_type
}

output "turbonomic_recommendation" {
  description = "The machine type recommended by Turbonomic (if available)"
  value       = data.turbonomic_google_compute_instance.example.new_machine_type
}

output "current_machine_type" {
  description = "The current GCP machine type (if exists)"
  value       = try(data.google_compute_instance.existing.machine_type, "N/A - New instance")
}
```

## Scenario Behavior

The fallback pattern behaves differently depending on whether the virtual machine already exists and whether Turbonomic recommendations are available.

### When Turbonomic returns a recommendation for an existing virtual machine:
1. Terraform uses the Turbonomic recommended instance type
2. Existing instance type is ignored

Example result: `t3.medium` (Turbonomic recommendation)

### When Turbonomic is temporarily unavailable or does not return a recommendation:
1. Terraform uses the current cloud provider instance type
2. No resize action occurs

This behavior helps prevent unintended infrastructure changes.

### When the virtual machine does not already exist:
1. Turbonomic recommendation is null
2. Existing cloud provider instance type is unavailable
3. Terraform uses the default instance type

Example result: `t2.nano` (default fallback)
