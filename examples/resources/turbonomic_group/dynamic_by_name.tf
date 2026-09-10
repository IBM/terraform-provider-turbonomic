# Dynamic group - all VMs whose name matches a regex
resource "turbonomic_group" "prod_vms_dynamic" {
  display_name     = "Prod VMs (dynamic)"
  group_type       = "VirtualMachine"
  is_static        = false
  logical_operator = "AND"

  criteria_list {
    filter_entity  = "vm"
    filter_field   = "name"
    operator       = "regex"
    value          = "^prod-.*"
    case_sensitive = false
  }
}
