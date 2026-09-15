data "netbox_ipsec_proposal" "by_name" {
  name = "ESP-AES256-SHA256"
}

# Lookup by arbitrary API filters (any query parameter of the list endpoint):
data "netbox_ipsec_proposal" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "ipsec_proposal_id" {
  value = data.netbox_ipsec_proposal.by_name.id
}
