# Static group - members defined by explicit UUID list
resource "turbonomic_group" "prod_vms_static" {
  display_name     = "Prod VMs"
  group_type       = "VirtualMachine"
  is_static        = true
  member_uuid_list = ["vm-uuid-1", "vm-uuid-2", "vm-uuid-3"]
}
