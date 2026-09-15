# Active config contexts scoped to a region.
data "netbox_config_contexts" "emea" {
  filters = [
    { name = "region", value = "emea" },
    { name = "is_active", value = "true" },
  ]
}

output "emea_contexts" {
  value = data.netbox_config_contexts.emea.items[*].name
}
