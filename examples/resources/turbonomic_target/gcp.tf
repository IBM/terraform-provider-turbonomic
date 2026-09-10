resource "turbonomic_target" "gcp" {
  type     = "GCP Service Account"
  category = "Public Cloud"

  input_fields = {
    name = "GCP Production"
  }

  input_fields_wo = {
    serviceAccountKey = file(var.gcp_credentials_file)
  }

  input_fields_wo_version = 1
}
