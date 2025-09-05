---
layout: ""
page_title: "Google Cloud examples"
description: |-
 This guide focuses on different google cloud resources examples.
---

# Google Cloud examples

This guide focuses on using Turbonomic data sources with [Google Cloud](https://registry.terraform.io/providers/hashicorp/google/latest/docs) resources, enabling dynamic resource configuration based on Turbonomic recommendations.

## Google compute engine example

The `machine_type` is set to use the `turbonomic_google_compute_instance` data source unless null is returned, in which case it uses `<default_machine_type>` by default.

```terraform
provider "google" {
  project = "example-turbonomic-terraform"
  region  = "us-central1"
}

data "turbonomic_google_compute_instance" "example" {
  entity_name          = "<entity_name>"
  default_machine_type = "<default_machine_type>"
}

resource "google_compute_instance" "exampleVirtualMachine" {
  name         = "exampleVirtualMachine"
  machine_type = data.turbonomic_google_compute_instance.example.new_machine_type
  zone         = "us-central1-a"
  labels       = provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider

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

## Google persistent disk example

The Google Persistent Disk resource is confgured to use the `turbonomic_google_compute_disk` data source unless null is returned, in which case it uses `<default_type>` by default.

```terraform
provider "google" {
  project = "example-turbonomic-terraform"
  region  = "us-central1"
}

data "turbonomic_google_compute_disk" "example" {
  entity_name                    = "<entity_name>"
  default_type                   = "<default_type>"
  default_provisioned_iops       = var.default_provisioned_iops
  default_provisioned_throughput = var.default_provisioned_throughput
  default_size                   = var.default_size
}

resource "google_compute_disk" "default" {
  name                   = "example-gcp-data-disk"
  type                   = data.turbonomic_google_compute_disk.example.default_type
  provisioned_iops       = data.turbonomic_google_compute_disk.example.new_provisioned_iops
  provisioned_throughput = data.turbonomic_google_compute_disk.example.new_provisioned_throughput
  size                   = data.turbonomic_google_compute_disk.example.new_size
  zone                   = "us-central1-a"
  image                  = "debian-11-bullseye-v20220719"
  labels = merge(
    {
      environment = "dev"
    },
    provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider
  )
  physical_block_size_bytes = 4096
}
```
