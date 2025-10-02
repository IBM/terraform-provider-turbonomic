provider "google" {
  project = "example-turbonomic-terraform"
  region  = "us-central1"
}

data "turbonomic_google_compute_instance" "example" {
  entity_name          = "<entity_name>"
  default_machine_type = "<default_machine_type>"
  vendor_id            = "<vendor_id>"
}

resource "google_compute_instance" "example_compute_instancee" {
  name         = "<entity_name>"
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
