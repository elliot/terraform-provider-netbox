data "netbox_custom_links" "enabled" {
  filters = [{ name = "enabled", value = "true" }]
}

output "enabled_custom_links" {
  value = data.netbox_custom_links.enabled.items[*].name
}
