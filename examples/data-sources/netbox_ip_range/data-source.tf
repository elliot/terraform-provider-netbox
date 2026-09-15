data "netbox_ip_range" "example" {
  start_address = "10.10.20.100/24"
}

# Any API filter of /api/ipam/ip-ranges/ works; the lookup must match exactly one object.
data "netbox_ip_range" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "ip_range_id" {
  value = data.netbox_ip_range.example.id
}
