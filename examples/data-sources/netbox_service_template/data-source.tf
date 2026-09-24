data "netbox_service_template" "example" {
  name = "HTTPS"
}

# Any API filter of /api/ipam/service-templates/ works; the lookup must match exactly one object.
data "netbox_service_template" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "service_template_id" {
  value = data.netbox_service_template.example.id
}
