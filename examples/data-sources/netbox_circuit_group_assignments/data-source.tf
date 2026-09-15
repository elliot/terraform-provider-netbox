# Every circuit in a group, with its priority.
data "netbox_circuit_group_assignments" "uplinks" {
  filters = [
    { name = "group", value = "ams-uplinks" },
  ]
}

output "uplink_members" {
  value = { for a in data.netbox_circuit_group_assignments.uplinks.items : a.member_id => a.priority }
}
