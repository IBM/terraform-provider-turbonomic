data "turbonomic_google_compute_disk" "example" {
  entity_name  = "exampleVirtualVolume"
  default_type = "pd-standard"
}
