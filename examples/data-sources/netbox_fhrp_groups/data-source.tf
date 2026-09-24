data "netbox_fhrp_groups" "all" {}

data "netbox_fhrp_groups" "filtered" {
  filters = [
    { name = "protocol", value = "vrrp3" },
  ]
  limit = 50
}

output "fhrp_groups_ids" {
  value = data.netbox_fhrp_groups.filtered.items[*].id
}
