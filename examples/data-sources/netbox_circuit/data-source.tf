# Circuits have no natural key besides the provider-scoped cid, so look them
# up by ID or by filter.
data "netbox_circuit" "by_id" {
  id = 1001
}

data "netbox_circuit" "by_cid" {
  filters = [
    { name = "provider", value = "lumen" },
    { name = "cid", value = "LUM-DIA-48213" },
  ]
}

output "ams_dia_status" {
  value = data.netbox_circuit.by_cid.status
}
