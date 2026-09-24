data "netbox_tunnel_groups" "all" {}

data "netbox_tunnel_groups" "filtered" {
  filters = [
    { name = "q", value = "branch" },
  ]
  limit = 50
}

output "tunnel_group_slugs" {
  value = data.netbox_tunnel_groups.filtered.items[*].slug
}
