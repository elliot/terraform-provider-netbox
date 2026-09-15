data "netbox_circuit_type" "dia" {
  slug = "internet-access"
}

resource "netbox_circuit" "example" {
  cid         = "LUM-DIA-48213"
  provider_id = netbox_provider.example.id
  type_id     = data.netbox_circuit_type.dia.id
}
