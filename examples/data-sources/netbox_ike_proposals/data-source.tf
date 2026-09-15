data "netbox_ike_proposals" "all" {}

data "netbox_ike_proposals" "filtered" {
  filters = [
    { name = "authentication_method", value = "preshared-keys" },
  ]
  limit = 50
}

output "ike_proposals_names" {
  value = data.netbox_ike_proposals.filtered.items[*].name
}
