data "netbox_circuit_termination" "a_side" {
  filters = [
    { name = "circuit_id", value = tostring(netbox_circuit.example.id) },
    { name = "term_side", value = "A" },
  ]
}

output "a_side_xconnect" {
  value = data.netbox_circuit_termination.a_side.xconnect_id
}
