data "netbox_circuit_groups" "acme" {
  filters = [
    { name = "tenant", value = "acme-corp" },
  ]
}

output "acme_circuit_groups" {
  value = data.netbox_circuit_groups.acme.items[*].name
}
