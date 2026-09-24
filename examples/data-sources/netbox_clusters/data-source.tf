# All clusters (paginated by NetBox; use limit to cap the result).
data "netbox_clusters" "all" {}

# Filters take any query parameter of /api/virtualization/clusters/.
data "netbox_clusters" "filtered" {
  filters = [
    { name = "type", value = "vmware-vsphere" },
  ]
  limit = 50
}

output "clusters_ids" {
  value = data.netbox_clusters.filtered.items[*].id
}
