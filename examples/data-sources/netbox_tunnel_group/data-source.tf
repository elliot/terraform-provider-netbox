data "netbox_tunnel_group" "by_slug" {
  slug = "branch-vpn"
}

# Lookup by arbitrary API filters (any query parameter of the list endpoint):
data "netbox_tunnel_group" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "tunnel_group_id" {
  value = data.netbox_tunnel_group.by_slug.id
}
