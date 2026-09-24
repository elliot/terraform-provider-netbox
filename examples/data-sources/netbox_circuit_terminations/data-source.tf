data "netbox_circuit_terminations" "at_site" {
  filters = [
    { name = "site", value = "ams-office" },
  ]
}

output "circuit_ids_at_site" {
  value = distinct(data.netbox_circuit_terminations.at_site.items[*].circuit_id)
}
