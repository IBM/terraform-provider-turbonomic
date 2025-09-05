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
