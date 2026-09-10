---
page_title: "Turbonomic as Code  - Policy and Schedule Management"
description: |-
  This guide shows how to combine turbonomic_group, turbonomic_schedule, turbonomic_placement_policy, turbonomic_settings_policy, and turbonomic_user data sources and resources to reproduce the Turbonomic web UI policy workflow entirely in Terraform.
---

# Turbonomic as Code  - Policy and schedule management

The Turbonomic web UI guides you through a multi-step workflow when you create or manage policies:

1. **Choose a scope**  - pick the groups of entities the policy applies to
2. **Attach a schedule**  - optionally restrict when actions can execute
3. **Configure settings**  - set action automation modes and resource thresholds
4. **Review ownership**  - understand who manages this policy

This guide shows how each of those steps maps to a Terraform resource or data source, and how to compose them into a single module that is fully reproducible and auditable.

---

## The full workflow

```
turbonomic_group (resource)           ← create or look up entity groups
         │
         ▼
turbonomic_schedule (data source)     ← look up an existing schedule by name
         │
         ▼
turbonomic_settings_policy (resource) ← create the automation policy, referencing group + schedule
         │
turbonomic_placement_policy (resource)← optionally constrain where VMs can run
         │
turbonomic_user (data source)         ← audit who has the role that manages these resources
```

---

## Step 1  - Define or look up the entity groups

In the Turbonomic UI you start by selecting a scope. In Terraform you either create the group or look it up.

### Create a new dynamic group

```terraform
# All VMs whose names start with "prod-"  - matches UI dynamic group criteria
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

# All hosts in the production cluster
resource "turbonomic_group" "prod_hosts" {
  display_name     = "Prod Hosts"
  group_type       = "PhysicalMachine"
  is_static        = false
  logical_operator = "AND"

  criteria_list {
    filter_entity = "pm"
    filter_field  = "cluster"
    operator      = "equals"
    value         = "Prod-Cluster"
  }
}
```

### Or look up an existing group

```terraform
# Reference a group that already exists in Turbonomic (created via UI or discovered)
data "turbonomic_group" "existing_prod_vms" {
  display_name = "Prod VMs"
}

locals {
  # Use the looked-up group UUID if it exists, otherwise fall back to the created group
  prod_vm_group_uuid = (
    length(data.turbonomic_group.existing_prod_vms.groups) > 0
    ? data.turbonomic_group.existing_prod_vms.groups[0].uuid
    : turbonomic_group.prod_vms.uuid
  )
}
```

---

## Step 2  - Create or look up the action execution schedule

In the Turbonomic UI you choose a schedule from a dropdown. In Terraform you can create one with the `turbonomic_schedule` resource, or look up an existing one with the data source.

### Create a new schedule

```terraform
resource "turbonomic_schedule" "maintenance" {
  display_name = "Weekend Maintenance Window"
  start_date   = "2025-01-01T00:00"
  start_time   = "2000-01-01T22:00"
  end_time     = "2000-01-02T06:00"
  time_zone    = "UTC"

  recurrence = {
    type         = "WEEKLY"
    days_of_week = ["Sat", "Sun"]
  }
}
```

### Or look up an existing schedule

```terraform
data "turbonomic_schedule" "maintenance" {
  display_name = "Maintenance Window"
}

locals {
  # Safely extract the UUID  - will be null if no schedule matches
  maintenance_schedule_uuid = (
    length(data.turbonomic_schedule.maintenance.schedules) > 0
    ? data.turbonomic_schedule.maintenance.schedules[0].uuid
    : null
  )
}
```

---

## Step 3  - Create the settings (automation) policy

This is the core of the UI "Automation Policy" form. You specify the entity type, scope groups, schedule, and per-setting overrides.

```terraform
resource "turbonomic_settings_policy" "prod_vm_automation" {
  name        = "prod-vm-automation-policy"
  entity_type = "VirtualMachine"
  disabled    = false

  # Scope to the group created/looked up in Step 1
  scope_uuids = [local.prod_vm_group_uuid]

  # Attach the schedule from Step 2 (null = no schedule restriction)
  schedule_uuid = turbonomic_schedule.maintenance.uuid

  note = "Managed by Terraform  - mirrors Prod VM policy in UI"

  # Automation mode overrides  - mirrors the "Automation" tab in the UI policy form
  settings_managers {
    category = "automationmanager"

    # VM moves: automatic during maintenance window
    settings {
      uuid  = "moveVirtualMachine"
      value = "AUTOMATIC"
    }

    # VM resize: recommend only (human approval required)
    settings {
      uuid  = "resizeVirtualMachine"
      value = "RECOMMEND"
    }

    # VM suspension: disabled
    settings {
      uuid  = "suspendVirtualMachine"
      value = "DISABLED"
    }
  }

  # Utilisation thresholds  - mirrors the "Constraints" tab in the UI
  settings_managers {
    category = "marketsettingsmanager"

    # Target CPU utilisation: 70%
    settings {
      uuid  = "resizeTargetUtilizationVcpu"
      value = "70.0"
    }

    # Target memory utilisation: 80%
    settings {
      uuid  = "resizeTargetUtilizationVmem"
      value = "80.0"
    }
  }
}
```

---

## Step 4  - Create a placement policy (optional)

In the UI this is the "Placement Policy" section. Use it to bind VMs to specific hosts or prevent co-location.

```terraform
# BIND_TO_GROUP  - prod VMs must only run on prod hosts
resource "turbonomic_placement_policy" "bind_prod_vms" {
  name             = "bind-prod-vms-to-prod-hosts"
  type             = "BIND_TO_GROUP"
  enabled          = true
  buyer_group_uuid = local.prod_vm_group_uuid
  seller_group_uuid = turbonomic_group.prod_hosts.uuid
}
```

```terraform
# MUST_NOT_RUN_TOGETHER  - two VM groups must not share a host
resource "turbonomic_group" "db_vms" {
  display_name     = "DB VMs"
  group_type       = "VirtualMachine"
  is_static        = false
  logical_operator = "AND"

  criteria_list {
    filter_entity  = "vm"
    filter_field   = "name"
    operator       = "regex"
    value          = "^db-"
    case_sensitive = false
  }
}

resource "turbonomic_placement_policy" "separate_app_db" {
  name             = "separate-app-and-db-vms"
  type             = "MUST_NOT_RUN_TOGETHER"
  enabled          = true
  buyer_group_uuid  = local.prod_vm_group_uuid
  seller_group_uuid = turbonomic_group.db_vms.uuid
}
```

---

## Step 5  - Manage or audit who manages these policies

In the UI you can see who created or last modified a policy. In Terraform you can create users with the `turbonomic_user` resource and audit existing accounts with the data source.

```terraform
# Create a scoped observer user for this environment
resource "turbonomic_user" "policy_observer" {
  username          = "ops-observer"
  password          = var.observer_password # pragma: allowlist secret
  role_names        = ["OBSERVER"]
  scope_group_uuids = [turbonomic_group.prod_vms.uuid]
}

# Audit: look up all ADMINISTRATOR-role users
data "turbonomic_user" "admins" {
  role = "ADMINISTRATOR"
}

output "admin_count" {
  description = "Number of ADMINISTRATOR accounts in Turbonomic"
  value       = length(data.turbonomic_user.admins.users)
}

output "admin_usernames" {
  description = "All administrator usernames"
  value       = [for u in data.turbonomic_user.admins.users : u.username]
}
```

---

## Step 6  - Attach a workflow (optional)

Workflows trigger external integrations when an action is executed. Use `turbonomic_workflow` to create a WEBHOOK that fires before a resize is applied.

```terraform
resource "turbonomic_workflow" "resize_approval" {
  display_name = "Resize Approval Webhook"
  type         = "WEBHOOK"
  action_type  = "RESIZE"
  action_phase = "PRE"
  entity_type  = "VirtualMachine"

  webhook_url    = var.approval_webhook_url
  webhook_method = "POST"
  webhook_template = jsonencode({
    action = "{{action.actionType}}"
    entity = "{{target.displayName}}"
    risk   = "{{action.risk}}"
  })
}
```

---

## Complete end-to-end example

The following is a self-contained configuration that creates a group, creates a schedule, creates both a settings policy and a placement policy, and audits the owning administrator  - the full UI workflow in a single Terraform module.

```terraform
# ── variables ──────────────────────────────────────────────────────────────────

variable "policy_owner_username" {
  description = "Turbonomic username expected to own these policies"
  type        = string
}

variable "schedule_name" {
  description = "Display name of the Turbonomic schedule to attach to the automation policy"
  type        = string
  default     = "Maintenance Window"
}

# ── groups ─────────────────────────────────────────────────────────────────────

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

resource "turbonomic_group" "prod_hosts" {
  display_name     = "Prod Hosts"
  group_type       = "PhysicalMachine"
  is_static        = false
  logical_operator = "AND"

  criteria_list {
    filter_entity = "pm"
    filter_field  = "cluster"
    operator      = "equals"
    value         = "Prod-Cluster"
  }
}

# ── schedule ───────────────────────────────────────────────────────────────────

resource "turbonomic_schedule" "maintenance" {
  display_name = "Weekend Maintenance Window"
  start_date   = "2025-01-01T00:00"
  start_time   = "2000-01-01T22:00"
  end_time     = "2000-01-02T06:00"
  time_zone    = "UTC"

  recurrence = {
    type         = "WEEKLY"
    days_of_week = ["Sat", "Sun"]
  }
}

# ── settings policy ────────────────────────────────────────────────────────────

resource "turbonomic_settings_policy" "prod_automation" {
  name          = "prod-vm-automation-policy"
  entity_type   = "VirtualMachine"
  disabled      = false
  scope_uuids   = [turbonomic_group.prod_vms.uuid]
  schedule_uuid = turbonomic_schedule.maintenance.uuid
  note          = "Managed by Terraform"

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

  settings_managers {
    category = "marketsettingsmanager"
    settings {
      uuid  = "resizeTargetUtilizationVcpu"
      value = "70.0"
    }
    settings {
      uuid  = "resizeTargetUtilizationVmem"
      value = "80.0"
    }
  }
}

# ── placement policy ───────────────────────────────────────────────────────────

resource "turbonomic_placement_policy" "bind_prod_vms" {
  name              = "bind-prod-vms-to-prod-hosts"
  type              = "BIND_TO_GROUP"
  enabled           = true
  buyer_group_uuid  = turbonomic_group.prod_vms.uuid
  seller_group_uuid = turbonomic_group.prod_hosts.uuid
}

# ── user audit ─────────────────────────────────────────────────────────────────

data "turbonomic_user" "admins" {
  role = "ADMINISTRATOR"
}

data "turbonomic_user" "owner" {
  username = var.policy_owner_username
}

# ── outputs ────────────────────────────────────────────────────────────────────

output "settings_policy_uuid" {
  description = "UUID of the automation policy"
  value       = turbonomic_settings_policy.prod_automation.uuid
}

output "placement_policy_uuid" {
  description = "UUID of the placement policy"
  value       = turbonomic_placement_policy.bind_prod_vms.uuid
}

output "group_uuids" {
  description = "UUIDs of the created groups"
  value = {
    prod_vms   = turbonomic_group.prod_vms.uuid
    prod_hosts = turbonomic_group.prod_hosts.uuid
  }
}

output "schedule_uuid" {
  description = "UUID of the schedule attached to the automation policy"
  value       = turbonomic_schedule.maintenance.uuid
}

output "policy_owner_exists" {
  description = "Whether the expected policy owner account exists in Turbonomic"
  value       = length(data.turbonomic_user.owner.users) > 0
}

output "admin_usernames" {
  description = "All Turbonomic administrator usernames"
  value       = [for u in data.turbonomic_user.admins.users : u.username]
}
```

---

## Notes

- **Dependency ordering.** Terraform automatically orders operations: groups are created before policies that reference them. Use `depends_on` only if you need to enforce ordering between resources that do not have implicit references.
- **`scope_uuids` is optional.** When omitted or empty, the settings policy applies globally to all entities of the specified `entity_type`. This mirrors the "No Scope" option in the UI.
- **`schedule_uuid` is optional.** When omitted, actions execute at any time within the action mode constraints. When set, actions are further gated by the schedule window.
- **Placement policy `ModifyPlan` validation.** The provider validates at plan time that `MERGE` policies provide `merge_uuids` and omit `buyer_group_uuid`/`seller_group_uuid`, and that non-MERGE policies provide both `buyer_group_uuid` and `seller_group_uuid`. Errors appear in `terraform plan` before any API call is made.
- **`turbonomic_user` requires elevated privileges.** The `GET /api/v3/users` endpoint requires `ADMINISTRATOR` or `SITE_ADMIN` role. If your Terraform service account has a lower role, remove the `turbonomic_user` data sources from your configuration.
- **Read-only computed fields.** Fields such as `uuid`, `id`, `default`, `read_only` on settings policies and `uuid`, `environment_type`, `cloud_type` on groups are populated by Turbonomic after creation. They are available for use in `output` blocks or as references in other resources immediately after `terraform apply`.

For a full mapping of every Turbonomic UI action to its Terraform equivalent, see the [UI to Terraform Reference](terminology_mapping.md).
