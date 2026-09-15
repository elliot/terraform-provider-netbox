data "netbox_ipsec_policies" "all" {}

data "netbox_ipsec_policies" "filtered" {
  filters = [
    { name = "pfs_group", value = "14" },
  ]
  limit = 50
}

output "ipsec_policies_names" {
  value = data.netbox_ipsec_policies.filtered.items[*].name
}
