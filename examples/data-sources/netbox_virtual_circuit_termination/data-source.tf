data "netbox_virtual_circuit_termination" "example" {
  id = 61
}

output "termination_role" {
  value = data.netbox_virtual_circuit_termination.example.role
}
