# Combine two filters with AND - entities must match ALL criteria
resource "turbonomic_filter" "prod_vms" {
  criteria_list {
    filter_entity = "vm"
    filter_field  = "name"
    operator      = "regex"
    value         = "^prod-"
  }
}

resource "turbonomic_filter" "linux_vms" {
  criteria_list {
    filter_entity = "vm"
    filter_field  = "guest_os"
    operator      = "equals"
    value         = "Linux"
  }
}

resource "turbonomic_group" "prod_linux_vms" {
  display_name     = "Production Linux VMs"
  group_type       = "VirtualMachine"
  logical_operator = "AND"

  dynamic "criteria_list" {
    for_each = concat(
      turbonomic_filter.prod_vms.criteria,
      turbonomic_filter.linux_vms.criteria,
    )
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
