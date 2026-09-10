resource "turbonomic_target" "instana" {
  type     = "Instana"
  category = "Applications and Databases"

  input_fields = {
    address  = var.instana_hostname
    apiToken = var.instana_api_token
  }

  input_fields_wo_version = 1
}
