data "netbox_aggregates" "all" {}

data "netbox_aggregates" "filtered" {
  filters = [
    { name = "family", value = "4" },
  ]
  limit = 50
}

output "aggregates_ids" {
  value = data.netbox_aggregates.filtered.items[*].id
}
