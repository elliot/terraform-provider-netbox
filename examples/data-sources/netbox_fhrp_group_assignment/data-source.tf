data "netbox_fhrp_group_assignment" "example" {
  id = 123
}

# Any API filter of /api/ipam/fhrp-group-assignments/ works; the lookup must match exactly one object.
data "netbox_fhrp_group_assignment" "filtered" {
  filters = [
    { name = "interface_type", value = "dcim.interface" },
    { name = "interface_id", value = "42" },
  ]
}

output "fhrp_group_assignment_id" {
  value = data.netbox_fhrp_group_assignment.example.id
}
