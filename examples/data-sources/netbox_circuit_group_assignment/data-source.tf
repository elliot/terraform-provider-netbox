data "netbox_circuit_group_assignment" "example" {
  id = 11
}

output "assignment_priority" {
  value = data.netbox_circuit_group_assignment.example.priority
}
