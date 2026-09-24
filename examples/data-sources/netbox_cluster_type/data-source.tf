# Look up a single cluster type by slug.
data "netbox_cluster_type" "example" {
  slug = "vmware-vsphere"
}

# Any API filter of /api/virtualization/cluster-types/ works with filters; the lookup must match exactly one object.
data "netbox_cluster_type" "filtered" {
  filters = [
    { name = "q", value = "vmware" },
  ]
}

output "cluster_type_id" {
  value = data.netbox_cluster_type.example.id
}
