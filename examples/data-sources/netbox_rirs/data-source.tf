data "netbox_rirs" "all" {}

data "netbox_rirs" "filtered" {
  filters = [
    { name = "is_private", value = "true" },
  ]
  limit = 50
}

output "rirs_ids" {
  value = data.netbox_rirs.filtered.items[*].id
}
