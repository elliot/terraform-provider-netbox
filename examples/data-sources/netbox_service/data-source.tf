data "netbox_service" "example" {
  name = "nginx"
}

# Any API filter of /api/ipam/services/ works; the lookup must match exactly one object.
data "netbox_service" "filtered" {
  filters = [
    { name = "device", value = "web1" },
    { name = "name", value = "nginx" },
  ]
}

output "service_id" {
  value = data.netbox_service.example.id
}
