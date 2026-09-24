data "netbox_route_target" "example" {
  name = "65000:100"
}

# Any API filter of /api/ipam/route-targets/ works; the lookup must match exactly one object.
data "netbox_route_target" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "route_target_id" {
  value = data.netbox_route_target.example.id
}
