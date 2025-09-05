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
