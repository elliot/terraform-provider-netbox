resource "netbox_provider" "example" {
  name = "Lumen"
  slug = "lumen"
}

resource "netbox_provider_network" "example" {
  provider_id = netbox_provider.example.id
  name        = "Lumen MPLS EU"
}

resource "netbox_virtual_circuit_type" "example" {
  name = "L2VPN"
  slug = "l2vpn"
}

resource "netbox_virtual_circuit" "example" {
  cid                 = "LUM-L2VPN-9001"
  provider_network_id = netbox_provider_network.example.id
  type_id             = netbox_virtual_circuit_type.example.id
  status              = "active"
  description         = "AMS <-> FRA pseudowire"
}
