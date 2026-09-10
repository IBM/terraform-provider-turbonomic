# Combine two filters with OR - entities that match ANY criterion
resource "turbonomic_filter" "aws_vms" {
  criteria_list {
    filter_entity = "vm"
    filter_field  = "cloud_provider"
    operator      = "equals"
    value         = "aws"
  }
}

resource "turbonomic_filter" "azure_vms" {
  criteria_list {
    filter_entity = "vm"
    filter_field  = "cloud_provider"
    operator      = "equals"
    value         = "azure"
  }
}

resource "turbonomic_group" "cloud_vms" {
  display_name     = "AWS or Azure VMs"
  group_type       = "VirtualMachine"
  logical_operator = "OR"

  dynamic "criteria_list" {
    for_each = concat(
      turbonomic_filter.aws_vms.criteria,
      turbonomic_filter.azure_vms.criteria,
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
