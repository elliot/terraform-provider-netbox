data "netbox_tunnel" "by_name" {
  name = "gre-branch01"
}

# Lookup by arbitrary API filters (any query parameter of the list endpoint):
data "netbox_tunnel" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "tunnel_id" {
  value = data.netbox_tunnel.by_name.id
}
