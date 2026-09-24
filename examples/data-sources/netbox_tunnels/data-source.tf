data "netbox_tunnels" "all" {}

data "netbox_tunnels" "filtered" {
  filters = [
    { name = "status", value = "active" },
  ]
  limit = 50
}

output "tunnels_names" {
  value = data.netbox_tunnels.filtered.items[*].name
}
