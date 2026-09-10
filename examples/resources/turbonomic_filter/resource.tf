# Standalone filter - define criteria once, use in any group
resource "turbonomic_filter" "prod_vms" {
  criteria_list {
    filter_entity = "vm"
    filter_field  = "name"
    operator      = "regex"
    value         = "^prod-"
  }
}

resource "turbonomic_group" "prod_vms" {
  display_name     = "Production VMs"
  group_type       = "VirtualMachine"
  logical_operator = "AND"

  dynamic "criteria_list" {
    for_each = turbonomic_filter.prod_vms.criteria
    content {
      filter_entity  = criteria_list.value.filter_entity
      filter_field   = criteria_list.value.filter_field
      filter_type    = criteria_list.value.filter_type
      operator       = criteria_list.value.operator
      value          = criteria_list.value.value
      case_sensitive = criteria_list.value.case_sensitive
    }
  }
}
