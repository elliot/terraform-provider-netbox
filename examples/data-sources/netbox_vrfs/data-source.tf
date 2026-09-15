data "netbox_vrfs" "all" {}

data "netbox_vrfs" "filtered" {
  filters = [
    { name = "enforce_unique", value = "true" },
  ]
  limit = 50
}

output "vrfs_ids" {
  value = data.netbox_vrfs.filtered.items[*].id
}
