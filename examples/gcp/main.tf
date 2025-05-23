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
