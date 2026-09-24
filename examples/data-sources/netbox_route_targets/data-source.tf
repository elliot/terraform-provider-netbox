data "netbox_route_targets" "all" {}

data "netbox_route_targets" "filtered" {
  filters = [
    { name = "name__isw", value = "65000:" },
  ]
  limit = 50
}

output "route_targets_ids" {
  value = data.netbox_route_targets.filtered.items[*].id
}
