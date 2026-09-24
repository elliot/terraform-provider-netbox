data "netbox_provider_network" "mpls_eu" {
  name = "Lumen MPLS EU"
}

# Use it as the Z side of a circuit:
resource "netbox_circuit_termination" "z" {
  circuit_id       = netbox_circuit.example.id
  term_side        = "Z"
  termination_type = "circuits.providernetwork"
  termination_id   = data.netbox_provider_network.mpls_eu.id
}
