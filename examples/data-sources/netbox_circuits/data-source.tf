# All active circuits terminating at a site.
data "netbox_circuits" "ams_active" {
  filters = [
    { name = "site", value = "ams-office" },
    { name = "status", value = "active" },
  ]
  limit = 100
}

output "ams_circuit_ids" {
  value = data.netbox_circuits.ams_active.items[*].cid
}
