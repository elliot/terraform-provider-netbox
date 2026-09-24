data "netbox_virtual_circuit_terminations" "pw9001" {
  filters = [
    { name = "virtual_circuit_id", value = tostring(netbox_virtual_circuit.example.id) },
  ]
}

output "pw9001_interfaces" {
  value = data.netbox_virtual_circuit_terminations.pw9001.items[*].interface_id
}
