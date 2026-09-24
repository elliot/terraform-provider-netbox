data "netbox_circuit_group" "uplinks" {
  slug = "ams-uplinks"
}

resource "netbox_circuit_group_assignment" "example" {
  group_id    = data.netbox_circuit_group.uplinks.id
  member_type = "circuits.circuit"
  member_id   = netbox_circuit.example.id
  priority    = "primary"
}
