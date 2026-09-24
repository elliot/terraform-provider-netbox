data "netbox_vlan_groups" "all" {}

data "netbox_vlan_groups" "filtered" {
  filters = [
    { name = "scope_type", value = "dcim.site" },
  ]
  limit = 50
}

output "vlan_groups_ids" {
  value = data.netbox_vlan_groups.filtered.items[*].id
}
