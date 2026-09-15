data "netbox_provider_networks" "lumen" {
  filters = [
    { name = "provider", value = "lumen" },
  ]
}

output "lumen_network_names" {
  value = data.netbox_provider_networks.lumen.items[*].name
}
