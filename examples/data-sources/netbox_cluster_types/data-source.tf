# All cluster types (paginated by NetBox; use limit to cap the result).
data "netbox_cluster_types" "all" {}

# Filters take any query parameter of /api/virtualization/cluster-types/.
data "netbox_cluster_types" "filtered" {
  filters = [
    { name = "q", value = "vmware" },
  ]
  limit = 50
}

output "cluster_types_ids" {
  value = data.netbox_cluster_types.filtered.items[*].id
}
