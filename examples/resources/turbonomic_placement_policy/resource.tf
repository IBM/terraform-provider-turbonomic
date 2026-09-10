
# Bind a group of VMs to a group of hosts (VMs can only run on those hosts)
resource "turbonomic_placement_policy" "bind_to_hosts" {
  name              = "bind-prod-vms-to-prod-hosts"
  type              = "BIND_TO_GROUP"
  enabled           = true
  buyer_group_uuid  = turbonomic_group.prod_vms.uuid
  seller_group_uuid = turbonomic_group.prod_hosts.uuid
}

# Prevent two groups of VMs from running on the same host
resource "turbonomic_placement_policy" "separate_vms" {
  name             = "separate-app-and-db-vms"
  type             = "MUST_NOT_RUN_TOGETHER"
  buyer_group_uuid = turbonomic_group.app_vms.uuid
}

# Merge two clusters so Turbonomic treats them as one
resource "turbonomic_placement_policy" "merge_clusters" {
  name        = "merge-dev-clusters"
  type        = "MERGE"
  merge_type  = "Cluster"
  merge_uuids = [turbonomic_group.cluster_a.uuid, turbonomic_group.cluster_b.uuid]
}
