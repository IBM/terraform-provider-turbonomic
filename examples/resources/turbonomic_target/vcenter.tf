resource "turbonomic_target" "vcenter" {
  type     = "vCenter"
  category = "Hypervisor"

  input_fields = {
    address  = var.vcenter_host
    username = var.vcenter_username
  }

  input_fields_wo = {
    password = var.vcenter_password
  }

  input_fields_wo_version = 1
}
