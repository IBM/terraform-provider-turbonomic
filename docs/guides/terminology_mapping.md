---
page_title: "UI to Terraform Reference"
description: |-
  Maps every Turbonomic web UI concept to its equivalent Terraform resource, data source, or attribute in the IBM Turbonomic provider.
---

# UI to Terraform Reference

Every Turbonomic web UI object that can be managed through the provider is listed here alongside its Terraform equivalent.

---

## Quick Reference Table

| Turbonomic UI Term | Terraform Equivalent | Type | Notes |
|---|---|---|---|
| Group | `turbonomic_group` | Resource + Data Source | Static (member list) or Dynamic (criteria-based) |
| Dynamic Group Criteria | `criteria_list` block | Block inside `turbonomic_group` | Use `filter_entity` + `filter_field` shorthand, or raw `filter_type` |
| Reusable Criteria Filter | `turbonomic_filter` | Resource (local state) | Local state only  - no API call; compose with `concat()` |
| Target / Probe | `turbonomic_target` | Resource + Data Source | One resource per target; credentials via write-only `input_fields_wo` |
| Placement Policy | `turbonomic_placement_policy` | Resource + Data Source | Controls where VMs can/cannot run relative to hosts or other VMs |
| Automation Policy (Settings) | `turbonomic_settings_policy` | Resource + Data Source | Sets action modes, utilisation thresholds, scaling limits per entity type |
| Schedule (Action Execution) | `turbonomic_schedule` | Resource + Data Source | Restricts when automation policy actions execute |
| User | `turbonomic_user` | Resource + Data Source | Manage local users, role assignments, scope groups; write-only password |
| Parking Policy | `turbonomic_parking_policy` | Resource | Suspends/powers off entities on a timespan schedule |
| Timespan Schedule | `turbonomic_timespan` | Data Source | Referenced by parking policies; distinct from action execution schedules |
| Role | `turbonomic_role` | Data Source | Read-only  - look up role names for user management |
| Probe Registration | `turbonomic_probe` | Data Source | Read-only  - check probe availability before creating targets |
| Workflow (Webhook / Action Script) | `turbonomic_workflow` | Resource + Data Source | Automate action execution via webhook; discovered workflows readable via data source |

---

## Web UI to Terraform Mapping Reference

The table below maps specific UI navigation paths and actions to the exact Terraform resource, data source, or attribute that replaces them.

| Turbonomic Web UI action | Terraform equivalent |
|---|---|
| **Groups -> New Group -> Static** | `turbonomic_group` resource with `is_static = true` + `member_uuid_list` |
| **Groups -> New Group -> Dynamic** | `turbonomic_group` resource with `is_static = false` + `criteria_list` blocks |
| **Groups -> Search / filter by name** | `turbonomic_group` data source with `display_name` filter |
| **Targets -> Add Target** | `turbonomic_target` resource with `type`, `category`, `input_fields`, `input_fields_wo` |
| **Targets -> Search targets** | `turbonomic_target` data source with optional `type` / `category` / `display_name` filters |
| **Targets -> Probe types available** | `turbonomic_probe` data source with optional `type` / `category` filters |
| **Policies -> New Placement Policy** | `turbonomic_placement_policy` resource with `type`, `buyer_group_uuid`, `seller_group_uuid` |
| **Policies -> Search Placement Policies** | `turbonomic_placement_policy` data source with `name` / `type` filter |
| **Policies -> New Automation Policy -> Entity Type** | `entity_type` on `turbonomic_settings_policy` |
| **Policies -> New Automation Policy -> Scope** | `scope_uuids` on `turbonomic_settings_policy` (list of group UUIDs) |
| **Policies -> New Automation Policy -> Schedule** | `schedule_uuid` on `turbonomic_settings_policy` (UUID from `turbonomic_schedule`) |
| **Policies -> New Automation Policy -> Action Mode** | `settings_managers` block, `uuid = "moveVirtualMachine"`, `value = "AUTOMATIC"` etc. |
| **Policies -> New Automation Policy -> Utilisation** | `settings_managers` block, `uuid = "resizeTargetUtilizationVcpu"` etc. |
| **Policies -> New Automation Policy -> Disable Policy** | `disabled = true` on `turbonomic_settings_policy` |
| **Policies -> Search Automation Policies** | `turbonomic_settings_policy` data source with `name` / `entity_type` filter |
| **Policies -> New Placement Policy -> BIND_TO_GROUP** | `turbonomic_placement_policy` resource with `type = "BIND_TO_GROUP"` |
| **Policies -> New Placement Policy -> MERGE** | `turbonomic_placement_policy` resource with `type = "MERGE"` + `merge_uuids` + `merge_type` |
| **Policies -> Parking Policies -> New** | `turbonomic_parking_policy` resource with `level`, `criteria_list`, `attach_schedule` |
| **Policies -> Parking Policies -> Attach Schedule** | `attach_schedule.timespan_schedule_uuid` (UUID from `turbonomic_timespan` data source) |
| **Policies -> Workflows -> New Webhook** | `turbonomic_workflow` resource with `type = "WEBHOOK"` |
| **Policies -> Workflows -> Search** | `turbonomic_workflow` data source with optional `type` / `display_name` filters |
| **Schedules -> Search by name** | `turbonomic_schedule` data source with `display_name` filter |
| **Schedules -> New Schedule** | `turbonomic_schedule` resource |
| **Schedules -> Timespan Schedules -> Search** | `turbonomic_timespan` data source with `display_name` filter |
| **Settings -> Users -> Filter by role** | `turbonomic_user` data source with `role` filter |
| **Settings -> Users -> Filter by username** | `turbonomic_user` data source with `username` filter |
| **Settings -> Users -> Filter by login provider** | `turbonomic_user` data source with `login_provider = "LDAP"` or `"LOCAL"` |
| **Settings -> Users -> Add User** | `turbonomic_user` resource with `username`, `password`, `role_names`, `scope_group_uuids` |
| **Settings -> Users -> Assign Role** | `role_names` list on `turbonomic_user` resource |
| **Settings -> Roles -> List all roles** | `turbonomic_role` data source |

---

## Concepts

### Scope

**In the UI:** When you create a policy, you select a "scope"  - the groups of entities the policy applies to.

**In Terraform:**
```terraform
resource "turbonomic_settings_policy" "example" {
  name        = "prod-vm-policy"
  entity_type = "VirtualMachine"
  scope_uuids = [turbonomic_group.prod_vms.uuid]
  # ...
}
```
`scope_uuids` accepts a list of group UUIDs. Use `turbonomic_group` (resource) to create new groups or look up existing ones with the `turbonomic_group` data source.

---

### Group Types

| UI Concept | `is_static` | Key Attributes |
|---|---|---|
| Static Group (manually selected members) | `true` | `member_uuid_list`  - list of entity UUIDs |
| Dynamic Group (rule-based membership) | `false` | `criteria_list` blocks + `logical_operator` |

```terraform
# Static group  - explicit member list
resource "turbonomic_group" "static_example" {
  display_name    = "Hand-picked VMs"
  group_type      = "VirtualMachine"
  is_static       = true
  member_uuid_list = ["uuid-1", "uuid-2"]
}

# Dynamic group  - criteria-based
resource "turbonomic_group" "dynamic_example" {
  display_name     = "Prod VMs"
  group_type       = "VirtualMachine"
  is_static        = false
  logical_operator = "AND"

  criteria_list {
    filter_entity = "vm"
    filter_field  = "name"
    operator      = "regex"
    value         = "^prod-"
  }
}
```

---

### Criteria / Filters

**In the UI:** Dynamic groups are defined by filter criteria (entity type, field, operator, value).

**In Terraform:** Each `criteria_list` block is one filter criterion. The `filter_entity` + `filter_field` shorthand resolves to an internal `filter_type` at plan time. See the [`turbonomic_group` schema reference](../resources/group.md#filter-shorthand-reference) for all supported combinations.

| UI Field | Terraform Attribute | Example |
|---|---|---|
| Entity Type (e.g. "Virtual Machine") | `filter_entity` | `"vm"`, `"pm"`, `"storage"` |
| Field (e.g. "Name", "Tag", "Guest OS") | `filter_field` | `"name"`, `"tag"`, `"guest_os"` |
| Operator | `operator` | `"equals"`, `"regex"`, `"not_equals"` |
| Value | `value` | `"^prod-"`, `"env=production"` |
| Case Sensitive | `case_sensitive` | `true` / `false` |
| Internal filter key | `filter_type` | `"vmsByName"`  - auto-resolved from shorthand |

---

### Placement Policy Types

| UI Policy Type | `type` Value | Required Attributes |
|---|---|---|
| Bind VMs to Hosts | `BIND_TO_GROUP` | `buyer_group_uuid`, `seller_group_uuid` |
| Separate VMs (must not run together) | `MUST_NOT_RUN_TOGETHER` | `buyer_group_uuid`, `seller_group_uuid` |
| Co-locate VMs (must run together) | `MUST_RUN_TOGETHER` | `buyer_group_uuid`, `seller_group_uuid` |
| Merge Clusters | `MERGE` | `merge_uuids`, `merge_type` |
| At Most N per Host | `AT_MOST_N` | `buyer_group_uuid`, `seller_group_uuid`, `capacity` |
| At Most N Bound | `AT_MOST_N_BOUND` | `buyer_group_uuid`, `seller_group_uuid`, `capacity` |
| Bind to Complementary Group | `BIND_TO_COMPLEMENTARY_GROUP` | `buyer_group_uuid`, `seller_group_uuid` |
| Exclusive Bind to Group | `EXCLUSIVE_BIND_TO_GROUP` | `buyer_group_uuid`, `seller_group_uuid` |

---

### Automation Policy  - Action Modes

**In the UI:** Each entity type has action settings under Settings -> Automation -> Action Modes.

**In Terraform:** Set via `settings_managers` blocks in `turbonomic_settings_policy`. The `uuid` identifies the setting; `value` is the mode string.

| UI Action Mode | `value` String |
|---|---|
| Recommend | `"RECOMMEND"` |
| Manual | `"MANUAL"` |
| Automatic | `"AUTOMATIC"` |
| Disabled | `"DISABLED"` |

```terraform
resource "turbonomic_settings_policy" "automation" {
  name        = "vm-action-modes"
  entity_type = "VirtualMachine"
  scope_uuids = [turbonomic_group.prod_vms.uuid]

  settings_managers {
    category = "automationmanager"

    settings {
      uuid  = "moveVirtualMachine"
      value = "AUTOMATIC"
    }
    settings {
      uuid  = "resizeVirtualMachine"
      value = "MANUAL"
    }
  }
}
```

---

### Automation Policy  - Settings Managers

**In the UI:** Settings are grouped into categories (Automation, Cloud Scaling, Storage, etc.).

**In Terraform:** Each category corresponds to a `settings_managers` block `category` value.

| UI Category | `category` Value |
|---|---|
| Automation (action modes) | `automationmanager` |
| Cloud Scaling / Utilisation | `marketsettingsmanager` |
| Control (resize constraints) | `controlmanager` |
| Storage | `storagesettingsmanager` |
| Entity Priorities | `entityprioritiesmanager` |
| HCI (Hyper-Converged) | `hcisettingsmanager` |
| Business Application | `busappsettingsmanager` |
| App Server | `appsrvsettingsmanager` |

---

### Schedule (Action Execution Schedule)

**In the UI:** Under Policies -> Schedules, you create a schedule that restricts when automated actions run.

**In Terraform:** Create a schedule with the `turbonomic_schedule` resource, or look up an existing one with the data source, then reference its UUID in a `turbonomic_settings_policy`.

```terraform
# Create a weekly maintenance schedule
resource "turbonomic_schedule" "maintenance" {
  display_name = "Weekend Maintenance Window"
  start_date   = "2025-01-01T00:00"
  start_time   = "2000-01-01T22:00"
  end_time     = "2000-01-02T06:00"
  time_zone    = "America/New_York"

  recurrence = {
    type        = "WEEKLY"
    days_of_week = ["Sat", "Sun"]
  }
}

resource "turbonomic_settings_policy" "with_schedule" {
  name          = "vm-weekend-automation"
  entity_type   = "VirtualMachine"
  schedule_uuid = turbonomic_schedule.maintenance.uuid
  # ...
}
```

---

### Target Input Fields

**In the UI:** When you add a target, you fill in probe-specific fields (hostname, username, password, etc.).

**In Terraform:** These map to `input_fields` (non-sensitive) and `input_fields_wo` (write-only / sensitive).

| UI Field Type | Terraform Attribute |
|---|---|
| Non-sensitive config (hostname, display name) | `input_fields = { key = "value" }` |
| Sensitive credentials (password, token) | `input_fields_wo = { key = "value" }`  - never stored in state |
| Credential rotation trigger | `input_fields_wo_version`  - increment to re-send credentials |

Use the `turbonomic_probe` data source to discover which probe types are available:
```terraform
data "turbonomic_probe" "all" {}
output "available_probe_types" {
  value = [for p in data.turbonomic_probe.all.probes : p.type]
}
```

---

### Parking Policy (Suspend / Power Off on Schedule)

**In the UI:** Under Policies -> Parking Policies, you configure policies that suspend or power off entities during defined time windows.

**In Terraform:** Use the `turbonomic_parking_policy` resource. Reference a timespan schedule UUID from the `turbonomic_timespan` data source.

```terraform
data "turbonomic_timespan" "office_hours" {
  display_name = "Office Hours"
}

resource "turbonomic_parking_policy" "dev_vms" {
  display_name = "Park Dev VMs Outside Office Hours"
  level        = "ACCOUNT"

  attach_schedule {
    timespan_schedule_uuid = data.turbonomic_timespan.office_hours.timespans[0].uuid
  }

  criteria_list {
    filter_type = "vmsByName"
    exp_type    = "RXEQ"
    exp_val     = "^dev-"
  }
}
```

| UI Concept | Terraform Attribute |
|---|---|
| Scope Level | `level`  - `GLOBAL`, `CLOUD_PROVIDER`, `ACCOUNT`, `RESOURCE_GROUP`, `DATACENTER`, `TARGET` |
| Entity Filters | `criteria_list` blocks  - `filter_type`, `exp_type`, `exp_val` |
| Attach Schedule | `attach_schedule.timespan_schedule_uuid` (from `turbonomic_timespan` data source) |
| Mark as Unparkable | `restrict_unparkable = true` |
| Priority | `priority`  - tie-break when multiple policies apply at the same level |

-> **Note** Parking policies reference a **timespan schedule** UUID (`turbonomic_timespan` data source), which is distinct from an action execution schedule (`turbonomic_schedule`).

---

### Workflow (Action Automation)

**In the UI:** Under Policies -> Workflows, you configure webhooks, ServiceNow integrations, or action scripts to trigger when Turbonomic executes an action.

**In Terraform:** Use the `turbonomic_workflow` resource for WEBHOOK type. Other discovered workflow types (ACTION_SCRIPT, SERVICENOW, KAFKA) are available via the `turbonomic_workflow` data source.

| UI Workflow Type | `type` Value | Terraform |
|---|---|---|
| Webhook | `WEBHOOK` | `turbonomic_workflow` resource |
| Action Script | `ACTION_SCRIPT` | Discovered  - use data source |
| ServiceNow | `SERVICENOW` | Discovered  - use data source |
| ActionStream Kafka | `ACTIONSTREAM_KAFKA` | Discovered  - use data source |

```terraform
resource "turbonomic_workflow" "resize_approval" {
  display_name = "Resize Approval Webhook"
  type         = "WEBHOOK"
  action_type  = "RESIZE"
  action_phase = "PRE"
  entity_type  = "VirtualMachine"

  webhook_url         = "https://webhook.example.com/approve"
  webhook_method      = "POST"
  webhook_auth_method = "BASIC"
  webhook_username    = "turbo-svc"
  webhook_password    = var.webhook_password # pragma: allowlist secret
}
```

---

### User Management

**In the UI:** Under Settings -> Users, you create and manage local user accounts, assign roles, and restrict scope.

**In Terraform:** Use the `turbonomic_user` resource. Look up available role names with the `turbonomic_role` data source.

```terraform
data "turbonomic_role" "observer" {
  name = "OBSERVER"
}

resource "turbonomic_user" "alice" {
  username          = "alice"
  password          = var.alice_password # pragma: allowlist secret
  role_names        = ["OBSERVER"]
  scope_group_uuids = [turbonomic_group.prod_vms.uuid]
}
```

-> **Note** `password` is write-only  - it is sent to the API on create/update but never stored in Terraform state.

---

## Common UI Workflows

### "Create a policy that auto-resizes prod VMs during maintenance windows"

| UI Step | Terraform |
|---|---|
| 1. Create a Group named "Prod VMs" with name filter `^prod-` | `turbonomic_group.prod_vms` with `criteria_list` |
| 2. Go to Policies -> Schedules, create "Maintenance Window" | `turbonomic_schedule` resource |
| 3. Go to Policies -> Automation Policies, create new for VirtualMachine | `turbonomic_settings_policy` resource |
| 4. Set scope to "Prod VMs" group | `scope_uuids = [turbonomic_group.prod_vms.uuid]` |
| 5. Set schedule to "Maintenance Window" | `schedule_uuid = turbonomic_schedule.maintenance.uuid` |
| 6. Set Resize action mode to Automatic | `settings_managers { category = "automationmanager" settings { uuid = "resizeVirtualMachine" value = "AUTOMATIC" } }` |

```terraform
resource "turbonomic_group" "prod_vms" {
  display_name     = "Prod VMs"
  group_type       = "VirtualMachine"
  is_static        = false
  logical_operator = "AND"

  criteria_list {
    filter_entity = "vm"
    filter_field  = "name"
    operator      = "regex"
    value         = "^prod-"
  }
}

resource "turbonomic_schedule" "maintenance" {
  display_name = "Maintenance Window"
  start_date   = "2025-01-01T00:00"
  start_time   = "2000-01-01T22:00"
  end_time     = "2000-01-02T06:00"
  time_zone    = "UTC"

  recurrence = {
    type         = "WEEKLY"
    days_of_week = ["Sat", "Sun"]
  }
}

resource "turbonomic_settings_policy" "prod_auto_resize" {
  name          = "prod-auto-resize"
  entity_type   = "VirtualMachine"
  scope_uuids   = [turbonomic_group.prod_vms.uuid]
  schedule_uuid = turbonomic_schedule.maintenance.uuid

  settings_managers {
    category = "automationmanager"

    settings {
      uuid  = "resizeVirtualMachine"
      value = "AUTOMATIC"
    }
  }
}
```

---

Consult the [What's New](whats_new.md) guide for features added in each release. See the [Policy and Schedule Management](policy_and_schedule_management.md) guide for an end-to-end workflow combining these resources.
