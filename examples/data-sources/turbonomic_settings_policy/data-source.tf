data "turbonomic_settings_policy" "all" {}

# Filter by name
data "turbonomic_settings_policy" "by_name" {
  name = "prod-vm-automation-policy"
}

# Filter by entity type
data "turbonomic_settings_policy" "vm_policies" {
  entity_type = "VirtualMachine"
}

# Filter by disabled state - only enabled policies
data "turbonomic_settings_policy" "active" {
  disabled = false
}

# Combine: active VM policies
data "turbonomic_settings_policy" "active_vm_policies" {
  entity_type = "VirtualMachine"
  disabled    = false
}

# Reference UUID from first matching policy
output "policy_uuid" {
  value = length(data.turbonomic_settings_policy.by_name.policies) > 0 ? data.turbonomic_settings_policy.by_name.policies[0].uuid : null
}
