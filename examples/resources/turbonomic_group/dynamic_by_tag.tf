# Dynamic group - all VMs tagged environment=production
resource "turbonomic_group" "tagged_vms" {
  display_name     = "Tagged Production VMs"
  group_type       = "VirtualMachine"
  is_static        = false
  logical_operator = "AND"

  criteria_list {
    filter_entity = "vm"
    filter_field  = "tag"
    operator      = "equals"
    value         = "environment=production"
  }
}
