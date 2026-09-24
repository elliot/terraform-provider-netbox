resource "netbox_provider" "example" {
  name = "Lumen"
  slug = "lumen"
}

resource "netbox_circuit_type" "example" {
  name = "Internet access"
  slug = "internet-access"
}

resource "netbox_circuit" "primary" {
  cid         = "LUM-DIA-48213"
  provider_id = netbox_provider.example.id
  type_id     = netbox_circuit_type.example.id
}

resource "netbox_circuit" "backup" {
  cid         = "LUM-DIA-48214"
  provider_id = netbox_provider.example.id
  type_id     = netbox_circuit_type.example.id
}

resource "netbox_circuit_group" "example" {
  name = "AMS uplinks"
  slug = "ams-uplinks"
}

resource "netbox_circuit_group_assignment" "primary" {
  group_id    = netbox_circuit_group.example.id
  member_type = "circuits.circuit"
  member_id   = netbox_circuit.primary.id
  priority    = "primary"
}

resource "netbox_circuit_group_assignment" "backup" {
  group_id    = netbox_circuit_group.example.id
  member_type = "circuits.circuit"
  member_id   = netbox_circuit.backup.id
  priority    = "secondary"
}
