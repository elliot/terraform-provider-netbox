data "netbox_ike_proposal" "by_name" {
  name = "IKE-AES256-SHA256-DH14"
}

# Lookup by arbitrary API filters (any query parameter of the list endpoint):
data "netbox_ike_proposal" "by_id" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "ike_proposal_id" {
  value = data.netbox_ike_proposal.by_name.id
}
