data "turbonomic_entity_actions" "example" {
  entity_name      = "example-doc-collection"
  entity_type      = "DocumentCollection"
  action_types     = ["SCALE"]
  environment_type = "CLOUD"
  states           = ["READY"]
}
