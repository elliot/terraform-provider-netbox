data "netbox_vlans" "all" {}

data "netbox_vlans" "filtered" {
  filters = [
    { name = "group", value = "dc1-vlans" },
  ]
  limit = 50
}

output "vlans_ids" {
  value = data.netbox_vlans.filtered.items[*].id
}
