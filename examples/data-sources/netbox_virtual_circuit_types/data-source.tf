data "netbox_virtual_circuit_types" "all" {}

output "virtual_circuit_types" {
  value = data.netbox_virtual_circuit_types.all.items[*].slug
}
