resource "netbox_site" "example" {
  name = "Amsterdam office"
  slug = "ams-office"
}

resource "netbox_provider" "example" {
  name = "Lumen"
  slug = "lumen"
}

resource "netbox_provider_network" "example" {
  provider_id = netbox_provider.example.id
  name        = "Lumen MPLS EU"
}

resource "netbox_circuit_type" "example" {
  name = "MPLS"
  slug = "mpls"
}

resource "netbox_circuit" "example" {
  cid         = "LUM-MPLS-77120"
  provider_id = netbox_provider.example.id
  type_id     = netbox_circuit_type.example.id
}

# A side: the customer premises.
resource "netbox_circuit_termination" "a" {
  circuit_id       = netbox_circuit.example.id
  term_side        = "A"
  termination_type = "dcim.site"
  termination_id   = netbox_site.example.id
  port_speed       = 1000000 # Kbps
  upstream_speed   = 1000000
  xconnect_id      = "XC-AMS-0142"
  pp_info          = "MMR PP03 ports 7-8"
}

# Z side: the provider's MPLS cloud.
resource "netbox_circuit_termination" "z" {
  circuit_id       = netbox_circuit.example.id
  term_side        = "Z"
  termination_type = "circuits.providernetwork"
  termination_id   = netbox_provider_network.example.id
}
