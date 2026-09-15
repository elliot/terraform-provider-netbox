data "netbox_ipsec_profile" "by_name" {
  name = "branch-site-to-site"
}

# Lookup by arbitrary API filters (any query parameter of the list endpoint):
data "netbox_ipsec_profile" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "ipsec_profile_id" {
  value = data.netbox_ipsec_profile.by_name.id
}
