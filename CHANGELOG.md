## 2.0.0

FEATURES:

- **New Resource:** `turbonomic_target` - Manage probe targets (cloud accounts, Kubernetes clusters, vCenter, Instana, ServiceNow, and more) with secure write-only credential fields.
- **New Resource:** `turbonomic_group` - Manage static or dynamic entity groups with rich filter criteria.
- **New Resource:** `turbonomic_schedule` - Create and manage action execution schedules (one-time, daily, weekly, monthly recurrence). Reference via `schedule_uuid` in `turbonomic_settings_policy`.
- **New Resource:** `turbonomic_user` - Manage local user accounts with role assignments and group scope. Write-only `password` attribute never stored in state.
- **New Resource:** `turbonomic_workflow` - Manage WEBHOOK workflows for pre/post/replace action automation. Write-only `webhook_password` never stored in state.
- **New Resource:** `turbonomic_parking_policy` - Suspend cloud workloads on a timespan schedule to reduce idle costs. References `turbonomic_timespan` data source.
- **New Resource:** `turbonomic_filter` - Local state-only resource for composable, reusable filter criteria. Combine with `concat()` into any `turbonomic_group`.
- **New Resource:** `turbonomic_placement_policy` - Manage VM placement constraints (BIND_TO_GROUP, MERGE, AT_MOST_N, MUST_NOT_RUN_TOGETHER, and more).
- **New Resource:** `turbonomic_settings_policy` - Manage automation policies: action modes, scaling limits, utilisation thresholds, scoped to groups with optional schedule.
- **New Data Source:** `turbonomic_target` - Look up existing targets by name, type, or category.
- **New Data Source:** `turbonomic_group` - Look up groups by display name or type.
- **New Data Source:** `turbonomic_schedule` - Look up existing action execution schedules by display name.
- **New Data Source:** `turbonomic_user` - List users filtered by username, role, or login provider. Requires ADMINISTRATOR or SITE_ADMIN privileges.
- **New Data Source:** `turbonomic_role` - List Turbonomic roles; filter by name.
- **New Data Source:** `turbonomic_probe` - List registered probe types; filter by type or category.
- **New Data Source:** `turbonomic_timespan` - List timespan (parking) schedules; filter by display name.
- **New Data Source:** `turbonomic_workflow` - List discovered workflows; filter by type or display name.
- **New Data Source:** `turbonomic_placement_policy` - Look up placement policies by name, type, or enabled state.
- **New Data Source:** `turbonomic_settings_policy` - Look up automation policies by name or entity type.
- **New Data Source:** `turbonomic_kubernetes_workload` - WorkloadController RESIZE (per-container CPU/memory) and SCALE (replica count) recommendations.
- **New Data Source:** `turbonomic_kubernetes_namespace` - RESIZE quota recommendations for CPU/memory limit and request quotas across a Kubernetes namespace.
- **New Data Source:** `turbonomic_kubernetes_pod` - MOVE action recommendations for Kubernetes pods, returning current and recommended destination node.
- **New Data Source:** `turbonomic_kubernetes_node` - PROVISION, SUSPEND, and RECONFIGURE action recommendations for Kubernetes nodes.
- **New Data Source:** `turbonomic_kubernetes_volume` - SCALE action recommendations for Kubernetes PersistentVolumeClaims, returning recommended storage size in GiB.
- **New Data Source:** `turbonomic_aws_eks_node_group` - Recommended node count for AWS EKS node groups.
- **New Data Source:** `turbonomic_azurerm_aks_node_pool` - Recommended node count for Azure AKS node pools.
- **New Data Source:** `turbonomic_google_gke_node_pool` - Recommended node count for Google GKE node pools.
- **New Data Source:** `turbonomic_aws_eks_cluster` - Smart Parking state recommendations for AWS EKS clusters.
- **New Data Source:** `turbonomic_azurerm_aks_cluster` - Smart Parking state recommendations for Azure AKS clusters.

## 1.11.0
FEATURES:

- Add `server_name` and `resource_group_name` optional parameters to `turbonomic_azurerm_mssql_database` data-source

## 1.10.0
NOTES:

- Add guide for using turbonomic fallback pattern

## 1.9.0
NOTES:

- Update provider to use Go 1.25.0
- Bump google.golang.org/grpc to v1.79.3

## 1.8.0
BUG FIXES:

- Remove required-together validation for volume data sources
- Add missing vendor_id suggestion on duplicate entity error message
- Add warning message for invalid entity in `turbonomic_entity_actions` datasource

## 1.7.0
FEATURES:

- The provider now honors automation policies and schedules defined in Turbonomic when returning optimal value.

BUG FIXES:

- Applied minor updates to documentation.
- Bump github.com/cloudflare/circl to version v1.6.1
- Bump github.com/cli/go-gh/v2 to version v2.12.1

## 1.6.0
FEATURES:
- Introduced `iops` and `allocated_storage` attributes to the `aws_db_instance` data source.
- Added `vendor_id` attribute to the following data sources for improved searching:
    - `turbonomic_aws_ebs_volume`
    - `turbonomic_aws_instance`
    - `turbonomic_azurerm_managed_disk`
    - `turbonomic_google_compute_disk`
    - `turbonomic_google_compute_instance`

BUG FIXES:
- Resolved casing normalization issues for `vm_size` values in Azure-related data sources.
- Fixed tagging issues in the `turbonomic_entity_actions` data source.
- Corrected configuration validators across multiple data sources.
- Applied minor updates to documentation.


## 1.5.0

FEATURES:
- Added `iops`,`throughput` and `size` attributes to `aws_ebs_volume` data source
- Added `disk_iops_read_write`,`disk_mbps_read_write` and `disk_size_gb` to `azurerm_managed_disk` data source
- Added `provisioned_iops`,`provisioned_throughput` and `size` attributes to `google_compute_disk` data source
- Added `turbonomic_entity_actions` generic
- Deprecated `turbonomic_cloud_entity_recommendation` data source
- **New Data Resources:** `turbonomic_aws_instance`, `azurerm_linux_virtual_machine`, `azurerm_windows_virtual_machine`, `google_compute_instance`

BUG FIXES:

- Fixed `azurerm_managed_disk` data source to return correct values for storage_account_type

NOTES:

- Refactored the data sources

## 1.3.0

FEATURES:

- Added support for `client_secret_post` as the `clientAuthenticationMethods` when using OAuth 2.0 authentication

NOTES:

- Added guide with examples to use Turbonomic provider with modules
- Reorganised the Turbonomic Provider Registry documentation.

## 1.2.0 (Beta Release)

FEATURES:

- **New Data Resource:** `turbonomic_aws_db_instance`
- **New Data Resource:** `turbonomic_azurerm_managed_disk`
- **New Data Resource:** `turbonomic_aws_ebs_volume`
- **New Data Resource:** `turbonomic_google_compute_disk`
- **New Function:** `provider::turbonomic::get_tag()`
- Added `ApiInfo` to send the basic metadata info to the go-client.

NOTES:

- Provider now requires `Terraform v1.8.5`
- Update provider to use `github.com/IBM/turbonomic-go-client-v1.2.0`

## 1.1.0 (Beta Release)

FEATURES:

- Added `default_size` field in data source block, which replaces the need for checking `new_instance_type` for null
- Added oAuth 2.0 support for authenticating to Turbonomic's API
  - See [Creating and authenticating an OAuth 2.0 client](https://www.ibm.com/docs/en/tarm/8.15.0?topic=cookbook-authenticating-oauth-20-clients-api#cookbook_administration_oauth_authentication__title__4) for details

NOTES:

- **provider:** Update provider to use Go `1.23.7`, `github.com/IBM/turbonomic-go-client-v1.1.0` and `golang.org/x/net-v0.36.0`

## 1.0.1 (Beta Release)

BUG FIXES:

- **data-source/turbonomic_cloud_entity_recommendation:** Fixed issue with `turbonomic_cloud_data_source` data source where an error is throw when specifying a entity that does not exist


## 1.0.0 (Beta Release)

FEATURES:

- **New Data Resource:** `turbonomic_cloud_data_source`
