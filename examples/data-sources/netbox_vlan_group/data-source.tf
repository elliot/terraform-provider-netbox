data "netbox_vlan_group" "example" {
  slug = "dc1-vlans"
}

# Any API filter of /api/ipam/vlan-groups/ works; the lookup must match exactly one object.
data "netbox_vlan_group" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "vlan_group_id" {
  value = data.netbox_vlan_group.example.id
}
