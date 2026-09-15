data "netbox_ipsec_policy" "by_name" {
  name = "IPsec-branch"
}

# Lookup by arbitrary API filters (any query parameter of the list endpoint):
data "netbox_ipsec_policy" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "ipsec_policy_id" {
  value = data.netbox_ipsec_policy.by_name.id
}
