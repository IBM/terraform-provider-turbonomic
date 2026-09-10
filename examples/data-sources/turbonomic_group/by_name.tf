# Look up a specific group by name to reference its UUID in other resources
data "turbonomic_group" "prod_vms" {
  display_name = "Prod VMs"
}

output "prod_vms_uuid" {
  value = one(data.turbonomic_group.prod_vms.groups).uuid
}
