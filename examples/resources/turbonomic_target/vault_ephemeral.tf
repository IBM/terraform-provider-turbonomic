# Ephemeral data source - value never stored in state
ephemeral "vault_kv_secret_v2" "hcp_credentials" {
  mount = "kv"
  name  = "turbonomic/terraform/hcp"
}

resource "turbonomic_target" "terraform_hcp_vault" {
  type     = "Terraform"
  category = "Infrastructure as Code"

  input_fields = {
    displayName                = "HCP Terraform"
    resourceType               = "HCP"
    hcpOrganizationName        = var.hcp_org_name
    terraformHostname          = "app.terraform.io"
    validateServerCertificates = "true"
  }

  # Sensitive fields using ephemeral values (NEVER stored in state)
  input_fields_wo = {
    hcpToken = tostring(ephemeral.vault_kv_secret_v2.hcp_credentials.data["token"])
  }

  # Increment to trigger re-send of credentials on next apply
  input_fields_wo_version = 1
}
