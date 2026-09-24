data "netbox_prefix" "example" {
  prefix = "10.10.0.0/16"
}

# Any API filter of /api/ipam/prefixes/ works; the lookup must match exactly one object.
data "netbox_prefix" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "prefix_id" {
  value = data.netbox_prefix.example.id
}
