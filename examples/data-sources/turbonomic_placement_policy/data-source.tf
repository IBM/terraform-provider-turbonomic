data "turbonomic_placement_policy" "all" {}

# Filter by name
data "turbonomic_placement_policy" "by_name" {
  name = "bind-prod-vms-to-prod-hosts"
}

# Filter by type
data "turbonomic_placement_policy" "merges" {
  type = "MERGE"
}

# Filter by enabled state
data "turbonomic_placement_policy" "active" {
  enabled = true
}

# Combine: enabled BIND_TO_GROUP policies
data "turbonomic_placement_policy" "active_bindings" {
  type    = "BIND_TO_GROUP"
  enabled = true
}

# Reference UUID from first matching policy
output "policy_uuid" {
  value = length(data.turbonomic_placement_policy.by_name.policies) > 0 ? data.turbonomic_placement_policy.by_name.policies[0].uuid : null
}
