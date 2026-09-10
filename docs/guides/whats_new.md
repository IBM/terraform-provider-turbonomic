---
page_title: "_What's New"
description: |-
 This guide explains new features that are part of this release
---

# What's New

## v2.0.0

Version 2.0.0 introduces ten new Kubernetes optimization data sources covering workload rightsizing, namespace quota management, node pool scaling, cluster Smart Parking, and native Kubernetes entity actions. It also introduces **Turbonomic-as-Code (TaC)**: manage all Turbonomic configuration directly in Terraform — targets, groups, policies, schedules, users, and workflows — without touching the web UI.

~> **Note** All Turbonomic-native resources and data sources are in PREVIEW and may have breaking changes before general availability.

### Kubernetes Optimization Data Sources

Ten new data sources bring Turbonomic optimization recommendations into Kubernetes and managed Kubernetes infrastructure workflows, covering workload rightsizing, node pool scaling, cluster parking, and native Kubernetes entity actions.

#### Kubernetes Workload Rightsizing

##### `turbonomic_kubernetes_workload`

Retrieves RESIZE (per-container CPU/memory limits and requests) and SCALE (replica count) recommendations for a Kubernetes WorkloadController — Deployments, StatefulSets, and similar controllers.

```terraform
data "turbonomic_kubernetes_workload" "api" {
  cluster   = "Kubernetes-mycluster"
  namespace = "production"
  name      = "api-server"

  default_replicas = 2
  default_containers = [
    {
      name                   = "api"
      default_cpu_request    = "200m"
      default_cpu_limit      = "500m"
      default_memory_request = "256Mi"
      default_memory_limit   = "512Mi"
    }
  ]
}
```

---

#### Kubernetes Namespace Quota Rightsizing

##### `turbonomic_kubernetes_namespace`

Retrieves RESIZE recommendations for Kubernetes namespace resource quotas. Turbonomic analyses aggregate resource consumption across pods in a namespace and recommends quota increases when headroom is insufficient. Quotas are never reduced below their current value.

```terraform
data "turbonomic_kubernetes_namespace" "app" {
  cluster = "Kubernetes-mycluster"
  name    = "production"

  default_cpu_limit_quota   = "2000"
  default_cpu_request_quota = "1000"
  default_mem_limit_quota   = "4096"
  default_mem_request_quota = "2048"
}

resource "kubernetes_resource_quota" "app" {
  metadata { name = "production" }
  spec {
    hard = {
      "limits.cpu"      = "${data.turbonomic_kubernetes_namespace.app.new_cpu_limit_quota}m"
      "requests.cpu"    = "${data.turbonomic_kubernetes_namespace.app.new_cpu_request_quota}m"
      "limits.memory"   = "${data.turbonomic_kubernetes_namespace.app.new_mem_limit_quota}Mi"
      "requests.memory" = "${data.turbonomic_kubernetes_namespace.app.new_mem_request_quota}Mi"
    }
  }
}
```

---

#### Kubernetes Node and Pod Actions

##### `turbonomic_kubernetes_pod`

Retrieves MOVE action recommendations for a Kubernetes pod. When Turbonomic identifies a better placement — for example, to relieve resource pressure on an overloaded node — this data source surfaces the recommended destination node.

```terraform
data "turbonomic_kubernetes_pod" "web" {
  cluster   = "Kubernetes-mycluster"
  namespace = "production"
  name      = "web-6d8f9b7c4-xk2pq"
}

output "pod_move_recommendation" {
  value = data.turbonomic_kubernetes_pod.web.new_node
  # Display name of the recommended destination node
}
```

##### `turbonomic_kubernetes_node`

Retrieves PROVISION, SUSPEND, or RECONFIGURE action recommendations for a Kubernetes node. Use this to drive node lifecycle decisions in response to Turbonomic's cluster-wide capacity analysis.

```terraform
data "turbonomic_kubernetes_node" "worker" {
  cluster = "Kubernetes-mycluster"
  name    = "ip-10-0-1-42.ec2.internal"
}

output "node_action" {
  value = data.turbonomic_kubernetes_node.worker.action_type
  # PROVISION, SUSPEND, RECONFIGURE, or empty when no action is pending
}
```

##### `turbonomic_kubernetes_volume`

Retrieves SCALE action recommendations for a Kubernetes PersistentVolumeClaim (PVC). PVCs are modeled as VirtualVolume entities discovered by a Kubernetes probe. When Turbonomic recommends increasing storage, this data source returns the recommended size in GiB.

```terraform
data "turbonomic_kubernetes_volume" "data_pvc" {
  cluster          = "Kubernetes-mycluster"
  name             = "pvc-0bab48cf-92c5-4e79-9dd8-62bd4d2fa634"
  default_size_gib = 10
}

output "pvc_recommended_size" {
  value = data.turbonomic_kubernetes_volume.data_pvc.new_size_gib
  # Recommended size in GiB, or default_size_gib when no action is pending
}
```

---

#### Node Pool Scaling (EKS, AKS, GKE)

Three new data sources return the recommended node count for a managed Kubernetes node pool, derived from PROVISION/SUSPEND actions on the pool's node VMs.

##### `turbonomic_aws_eks_node_group`

```terraform
data "turbonomic_aws_eks_node_group" "workers" {
  cluster_name         = "Kubernetes-myekscluster"
  node_group_name      = "workers"
  default_desired_size = 3
}
```

##### `turbonomic_azurerm_aks_node_pool`

```terraform
data "turbonomic_azurerm_aks_node_pool" "system" {
  cluster_name       = "Kubernetes-myakscluster"
  name               = "system"
  default_node_count = 2
}
```

##### `turbonomic_google_gke_node_pool`

```terraform
data "turbonomic_google_gke_node_pool" "default" {
  cluster_name       = "Kubernetes-mygkecluster"
  node_pool_name     = "default-pool"
  default_node_count = 3
}
```

---

#### Cluster Smart Parking (EKS, AKS)

Two new data sources expose Turbonomic Smart Parking state and schedule recommendations for managed Kubernetes clusters, allowing Terraform to stop and start clusters on a schedule.

##### `turbonomic_aws_eks_cluster`

```terraform
data "turbonomic_aws_eks_cluster" "dev" {
  name          = "Kubernetes-myekscluster"
  default_state = "RUNNING"
}

output "eks_desired_state" {
  value = data.turbonomic_aws_eks_cluster.dev.new_state
  # RUNNING or STOPPED based on Turbonomic's parking schedule
}
```

##### `turbonomic_azurerm_aks_cluster`

```terraform
data "turbonomic_azurerm_aks_cluster" "dev" {
  name          = "Kubernetes-myakscluster"
  default_state = "RUNNING"
}
```

---

### Turbonomic-as-Code (TaC)

#### New resources

| Resource | Description |
|---|---|
| `turbonomic_target` | Manage probe targets (cloud accounts, Kubernetes clusters, vCenter, and more) |
| `turbonomic_group` | Manage static or dynamic entity groups with filter criteria |
| `turbonomic_filter` | Local state-only resource for composable, reusable filter criteria; combine into any group with `concat()` |
| `turbonomic_placement_policy` | Manage VM placement constraints (BIND_TO_GROUP, MERGE, AT_MOST_N, MUST_NOT_RUN_TOGETHER, and more) |
| `turbonomic_settings_policy` | Manage automation policies: action modes, scaling limits, and utilisation thresholds scoped to groups |
| `turbonomic_schedule` | Create and manage action execution schedules; one-time, daily, weekly, and monthly recurrence |
| `turbonomic_user` | Manage local user accounts with role assignments and group scope; write-only password |
| `turbonomic_workflow` | Manage WEBHOOK workflows for pre/post/replace action automation; write-only credentials |
| `turbonomic_parking_policy` | Suspend cloud workloads on a timespan schedule to reduce idle costs |

#### New data sources

| Data Source | Description |
|---|---|
| `turbonomic_target` | Look up existing targets by name, type, or category |
| `turbonomic_group` | Look up groups by display name or type |
| `turbonomic_schedule` | Look up action execution schedules by display name |
| `turbonomic_user` | List users filtered by username, role, or login provider |
| `turbonomic_role` | List all Turbonomic roles; filter by name |
| `turbonomic_probe` | List available probe (target type) registrations; filter by type or category |
| `turbonomic_timespan` | List timespan (parking) schedules; filter by display name |
| `turbonomic_workflow` | List discovered workflows; filter by type or display name |
| `turbonomic_placement_policy` | Look up placement policies by name, type, or enabled state |
| `turbonomic_settings_policy` | Look up automation policies by name or entity type |

#### Full resource inventory

All resources and data sources currently available:

**Resources (9):**
`turbonomic_target`, `turbonomic_group`, `turbonomic_filter`, `turbonomic_placement_policy`, `turbonomic_settings_policy`, `turbonomic_schedule`, `turbonomic_user`, `turbonomic_workflow`, `turbonomic_parking_policy`

**Data Sources (31):**
`turbonomic_target`, `turbonomic_group`, `turbonomic_placement_policy`, `turbonomic_settings_policy`, `turbonomic_schedule`, `turbonomic_user`, `turbonomic_role`, `turbonomic_probe`, `turbonomic_timespan`, `turbonomic_workflow`, plus 21 cloud and Kubernetes entity data sources

See the [UI to Terraform Reference](terminology_mapping.md) for a full mapping of every Turbonomic web UI action to its Terraform equivalent.

See the [Policy and Schedule Management](policy_and_schedule_management.md) guide for an end-to-end workflow.

---

For full attribute reference see the data source documentation:
- [`turbonomic_kubernetes_workload`](../data-sources/kubernetes_workload.md)
- [`turbonomic_kubernetes_namespace`](../data-sources/kubernetes_namespace.md)
- [`turbonomic_kubernetes_pod`](../data-sources/kubernetes_pod.md)
- [`turbonomic_kubernetes_node`](../data-sources/kubernetes_node.md)
- [`turbonomic_kubernetes_volume`](../data-sources/kubernetes_volume.md)
- [`turbonomic_aws_eks_node_group`](../data-sources/aws_eks_node_group.md)
- [`turbonomic_azurerm_aks_node_pool`](../data-sources/azurerm_aks_node_pool.md)
- [`turbonomic_google_gke_node_pool`](../data-sources/google_gke_node_pool.md)
- [`turbonomic_aws_eks_cluster`](../data-sources/aws_eks_cluster.md)
- [`turbonomic_azurerm_aks_cluster`](../data-sources/azurerm_aks_cluster.md)
