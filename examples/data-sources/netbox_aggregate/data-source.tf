data "netbox_aggregate" "example" {
  prefix = "10.0.0.0/8"
}

# Any API filter of /api/ipam/aggregates/ works; the lookup must match exactly one object.
data "netbox_aggregate" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "aggregate_id" {
  value = data.netbox_aggregate.example.id
}
