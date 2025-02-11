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
