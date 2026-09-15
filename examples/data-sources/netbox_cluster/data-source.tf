# Look up a single cluster by name.
data "netbox_cluster" "example" {
  name = "fra1-prod-01"
}

# Any API filter of /api/virtualization/clusters/ works with filters; the lookup must match exactly one object.
data "netbox_cluster" "filtered" {
  filters = [
    { name = "type", value = "vmware-vsphere" },
  ]
}

output "cluster_id" {
  value = data.netbox_cluster.example.id
}
