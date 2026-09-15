data "netbox_l2vpn" "by_name" {
  name = "tenant-a-overlay"
}

# Lookup by arbitrary API filters (any query parameter of the list endpoint):
data "netbox_l2vpn" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "l2vpn_id" {
  value = data.netbox_l2vpn.by_name.id
}
