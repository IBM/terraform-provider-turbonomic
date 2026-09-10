resource "turbonomic_target" "servicenow" {
  type     = "ServiceNow"
  category = "IT Management"

  input_fields = {
    displayName   = "ServiceNow Production"
    targetAddress = var.servicenow_hostname
    username      = var.servicenow_username
  }

  input_fields_wo = {
    password = var.servicenow_password
  }

  input_fields_wo_version = 1
}
