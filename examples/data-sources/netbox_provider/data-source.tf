data "netbox_provider" "lumen" {
  slug = "lumen"
}

output "lumen_circuit_count" {
  value = data.netbox_provider.lumen.circuit_count
}
