# Look up a single cluster group by slug.
data "netbox_cluster_group" "example" {
  slug = "emea"
}

# Any API filter of /api/virtualization/cluster-groups/ works with filters; the lookup must match exactly one object.
data "netbox_cluster_group" "filtered" {
  filters = [
    { name = "q", value = "emea" },
  ]
}

output "cluster_group_id" {
  value = data.netbox_cluster_group.example.id
}
