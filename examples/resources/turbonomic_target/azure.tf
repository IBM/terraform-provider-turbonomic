resource "turbonomic_target" "azure" {
  type     = "Azure Service Principal"
  category = "Public Cloud"

  input_fields = {
    name   = "Azure Production"
    tenant = var.azure_tenant_id
    client = var.azure_client_id
  }

  input_fields_wo = {
    key = var.azure_client_secret
  }

  input_fields_wo_version = 1
}
