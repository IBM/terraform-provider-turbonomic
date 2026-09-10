# Look up all VirtualMachine groups
data "turbonomic_group" "vm_groups" {
  group_type = "VirtualMachine"
}

output "vm_group_names" {
  value = [for g in data.turbonomic_group.vm_groups.groups : g.display_name]
}
