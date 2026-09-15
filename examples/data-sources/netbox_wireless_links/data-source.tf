data "netbox_wireless_links" "planned" {
  filters = [{ name = "status", value = "planned" }]
}

output "planned_links" {
  value = data.netbox_wireless_links.planned.items[*].id
}
