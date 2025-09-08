data "turbonomic_entity_actions" "example" {
  entity_name      = "aws-scale-test"
  entity_type      = "VirtualMachine"
  action_types     = ["SCALE"]
  environment_type = "CLOUD"
  states           = ["READY"]
}
