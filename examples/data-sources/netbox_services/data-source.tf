data "netbox_services" "all" {}

data "netbox_services" "filtered" {
  filters = [
    { name = "port", value = "443" },
  ]
  limit = 50
}

output "services_ids" {
  value = data.netbox_services.filtered.items[*].id
}
