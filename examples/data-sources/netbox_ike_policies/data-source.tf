data "netbox_ike_policies" "all" {}

data "netbox_ike_policies" "filtered" {
  filters = [
    { name = "version", value = "2" },
  ]
  limit = 50
}

output "ike_policies_names" {
  value = data.netbox_ike_policies.filtered.items[*].name
}
