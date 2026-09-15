data "netbox_ike_policy" "by_name" {
  name = "IKEv2-PSK"
}

# Lookup by arbitrary API filters (any query parameter of the list endpoint):
data "netbox_ike_policy" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "ike_policy_id" {
  value = data.netbox_ike_policy.by_name.id
}
