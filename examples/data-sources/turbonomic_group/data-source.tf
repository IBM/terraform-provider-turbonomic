# Look up all groups
data "turbonomic_group" "all" {}

output "group_count" {
  value = length(data.turbonomic_group.all.groups)
}

output "group_names" {
  value = [for g in data.turbonomic_group.all.groups : g.display_name]
}
