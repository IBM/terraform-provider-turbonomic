---
page_title: "turbonomic"
description: |-
  This guide covers all Turbonomic-native resources and data sources for managing Turbonomic configuration entirely in Terraform (Turbonomic as Code). These features are in PREVIEW.
---

# Turbonomic resources and data sources

~> **Note** The Turbonomic-native resources and data sources listed here are in PREVIEW. They are functional and supported but may have breaking changes before general availability.

This guide covers managing Turbonomic configuration entirely in Terraform  - groups, policies, schedules, users, workflows, and parking policies  - without touching the Turbonomic web UI.

For a step-by-step walkthrough see the [Policy and Schedule Management](policy_and_schedule_management.md) guide.
For a full UI to Terraform mapping see the [UI to Terraform Reference](terminology_mapping.md).

---

## Resources

All 9 Turbonomic-native resources are available. Each creates and manages the corresponding object in the Turbonomic platform.

| Resource | Description |
|---|---|
| [`turbonomic_target`](../resources/target.md) | Manage probe targets (cloud accounts, Kubernetes clusters, vCenter, etc.) |
| [`turbonomic_group`](../resources/group.md) | Manage static or dynamic entity groups with filter criteria |
| [`turbonomic_filter`](../resources/filter.md) | Local state-only composable filter criteria; combine into groups with `concat()` |
| [`turbonomic_placement_policy`](../resources/placement_policy.md) | VM placement constraints (BIND_TO_GROUP, MERGE, AT_MOST_N, and more) |
| [`turbonomic_settings_policy`](../resources/settings_policy.md) | Automation policies: action modes, scaling limits, and utilisation thresholds |
| [`turbonomic_schedule`](../resources/schedule.md) | Action execution schedules: one-time, daily, weekly, and monthly recurrence |
| [`turbonomic_user`](../resources/user.md) | Local user accounts with role assignments and group scope |
| [`turbonomic_workflow`](../resources/workflow.md) | WEBHOOK workflows for pre/post/replace action automation |
| [`turbonomic_parking_policy`](../resources/parking_policy.md) | Suspend cloud workloads on a timespan schedule to reduce idle costs |

---

## Data sources

| Data Source | Description |
|---|---|
| [`turbonomic_target`](../data-sources/target.md) | Look up existing targets by name or type |
| [`turbonomic_group`](../data-sources/group.md) | Look up groups by display name or type |
| [`turbonomic_placement_policy`](../data-sources/placement_policy.md) | Look up placement policies by name, type, or enabled state |
| [`turbonomic_settings_policy`](../data-sources/settings_policy.md) | Look up automation policies by name or entity type |
| [`turbonomic_schedule`](../data-sources/schedule.md) | Look up action execution schedules by display name |
| [`turbonomic_user`](../data-sources/user.md) | List users filtered by username, role, or login provider |
| [`turbonomic_role`](../data-sources/role.md) | List Turbonomic roles; filter by name |
| [`turbonomic_probe`](../data-sources/probe.md) | List registered probe types; filter by type or category |
| [`turbonomic_timespan`](../data-sources/timespan.md) | List timespan (parking) schedules; filter by display name |
| [`turbonomic_workflow`](../data-sources/workflow.md) | List discovered workflows; filter by type or display name |
| [`turbonomic_entity_actions`](../data-sources/entity_actions.md) | Query pending actions across any entity type |

---

## Quick-start example

```terraform
# Create a dynamic group of production VMs
resource "turbonomic_group" "prod_vms" {
  display_name     = "Prod VMs"
  group_type       = "VirtualMachine"
  is_static        = false
  logical_operator = "AND"

  criteria_list {
    filter_entity  = "vm"
    filter_field   = "name"
    operator       = "regex"
    value          = "^prod-"
    case_sensitive = false
  }
}

# Create a weekly maintenance schedule
resource "turbonomic_schedule" "weekend" {
  display_name = "Weekend Maintenance"
  start_date   = "2025-01-01T00:00"
  start_time   = "2000-01-01T22:00"
  end_time     = "2000-01-02T06:00"
  time_zone    = "UTC"

  recurrence = {
    type         = "WEEKLY"
    days_of_week = ["Sat", "Sun"]
  }
}

# Automate VM moves during the maintenance window
resource "turbonomic_settings_policy" "prod_automation" {
  name          = "prod-vm-automation"
  entity_type   = "VirtualMachine"
  scope_uuids   = [turbonomic_group.prod_vms.uuid]
  schedule_uuid = turbonomic_schedule.weekend.uuid

  settings_managers {
    category = "automationmanager"
    settings {
      uuid  = "moveVirtualMachine"
      value = "AUTOMATIC"
    }
    settings {
      uuid  = "resizeVirtualMachine"
      value = "RECOMMEND"
    }
  }
}
```
