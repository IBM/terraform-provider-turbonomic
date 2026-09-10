resource "turbonomic_target" "terraform_hcp" {
  type     = "Terraform"
  category = "Infrastructure as Code"

  input_fields = {
    displayName                = "HCP Terraform"
    resourceType               = "HCP"
    hcpOrganizationName        = var.hcp_org_name
    terraformHostname          = "app.terraform.io"
    validateServerCertificates = "true"
  }

  input_fields_wo = {
    hcpToken = var.hcp_token
  }

  input_fields_wo_version = 1
}
