resource "turbonomic_user" "observer" {
  username   = "alice"
  password   = var.alice_password
  role_names = ["OBSERVER"]
}

resource "turbonomic_user" "scoped_automator" {
  username          = "bob"
  password          = var.bob_password
  display_name      = "Bob (Scoped Automator)"
  role_names        = ["AUTOMATOR"]
  scope_group_uuids = [turbonomic_group.prod_vms.uuid]
}
