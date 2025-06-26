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
