data "netbox_fhrp_group" "example" {
  name = "office-gateway"
}

# Any API filter of /api/ipam/fhrp-groups/ works; the lookup must match exactly one object.
data "netbox_fhrp_group" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "fhrp_group_id" {
  value = data.netbox_fhrp_group.example.id
}
