---
layout: ""
page_title: "Google Cloud Examples"
description: |-
 This guide focuses on different google cloud resources examples.
---
This guide focuses on using Turbonomic data sources with [Google Cloud](https://registry.terraform.io/providers/hashicorp/google/latest/docs) resources, enabling dynamic resource configuration based on Turbonomic recommendations.

## Google compute engine example

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

The Google Persistent Disk resource is confgured to use the `turbonomic_google_compute_disk` data source unless null is returned,
in which case it uses `pd-standard` by default.

```terraform
provider "google" {
  project = "example-turbonomic-terraform"
  region  = "us-central1"
}

data "turbonomic_google_compute_disk" "example" {
  entity_name  = "example-gcp-data-disk"
  default_type = "pd-standard"
}

resource "google_compute_disk" "default" {
  name  = "example-gcp-data-disk"
  type  = data.turbonomic_google_compute_disk.example.default_type
  zone  = "us-central1-a"
  image = "debian-11-bullseye-v20220719"
  labels = merge(
    {
      environment = "dev"
    },
    provider::turbonomic::get_tag() //tag the resource as optimized by Turbonomic provider
  )
  physical_block_size_bytes = 4096
}
```
