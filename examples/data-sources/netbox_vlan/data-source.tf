data "netbox_vlan" "example" {
  name = "SERVERS"
}

# Any API filter of /api/ipam/vlans/ works; the lookup must match exactly one object.
data "netbox_vlan" "filtered" {
  filters = [
    { name = "vid", value = "110" },
    { name = "group", value = "dc1-vlans" },
  ]
}

output "vlan_id" {
  value = data.netbox_vlan.example.id
}
