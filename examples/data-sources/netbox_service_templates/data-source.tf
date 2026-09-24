data "netbox_service_templates" "all" {}

data "netbox_service_templates" "filtered" {
  filters = [
    { name = "q", value = "HTTP" },
  ]
  limit = 50
}

output "service_templates_ids" {
  value = data.netbox_service_templates.filtered.items[*].id
}
