# All cluster groups (paginated by NetBox; use limit to cap the result).
data "netbox_cluster_groups" "all" {}

# Filters take any query parameter of /api/virtualization/cluster-groups/.
data "netbox_cluster_groups" "filtered" {
  filters = [
    { name = "q", value = "emea" },
  ]
  limit = 50
}

output "cluster_groups_ids" {
  value = data.netbox_cluster_groups.filtered.items[*].id
}
