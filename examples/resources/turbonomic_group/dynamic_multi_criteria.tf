# Dynamic group - VMs in a specific cluster AND matching a name pattern
resource "turbonomic_group" "cluster_vms" {
  display_name     = "Cluster A Prod VMs"
  group_type       = "VirtualMachine"
  is_static        = false
  logical_operator = "AND"

  criteria_list {
    filter_entity = "vm"
    filter_field  = "cluster"
    operator      = "equals"
    value         = "Cluster-A"
  }

  criteria_list {
    filter_entity  = "vm"
    filter_field   = "name"
    operator       = "regex"
    value          = "^prod-"
    case_sensitive = false
  }
}
