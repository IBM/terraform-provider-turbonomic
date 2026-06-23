provider "google" {
  project = "my-project"
  region  = "us-central1"
}

# Query existing GCP instance (if it exists)
data "google_compute_instance" "existing" {
  name = "my-instance"
  zone = "us-central1-a"
}

# Query Turbonomic for recommendations
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
