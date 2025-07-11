---
page_title: "turbonomic_cloud_entity_recommendation Data Source - IBM Turbonomic"
subcategory: ""
description: |-
  The following example demonstrates the syntax for the turbonomic_cloud_entity_recommendation data source.
---

# turbonomic_cloud_entity_recommendation (Data Source)

The following example demonstrates the syntax for the `turbonomic_cloud_entity_recommendation` data source.

## Example Usage

```terraform
data "turbonomic_cloud_entity_recommendation" "example" {
  entity_name  = "exampleVirtualMachine"
  entity_type  = "VirtualMachine"
  default_size = "defaultSize"
}
```
<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `entity_name` (String) name of the cloud entity
- `entity_type` (String) type of the cloud entity

### Optional

- `default_size` (String) default tier of the cloud entity

### Read-Only

- `current_instance_type` (String) current tier of the cloud entity
- `entity_uuid` (String) Turbonomic UUID of the cloud entity
- `new_instance_type` (String) recommended tier of the cloud entity
