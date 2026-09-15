data "netbox_circuit_types" "all" {}

output "circuit_types" {
  value = { for t in data.netbox_circuit_types.all.items : t.slug => t.circuit_count }
}
