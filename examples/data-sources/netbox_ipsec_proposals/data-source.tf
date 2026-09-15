data "netbox_ipsec_proposals" "all" {}

data "netbox_ipsec_proposals" "filtered" {
  filters = [
    { name = "encryption_algorithm", value = "aes-256-cbc" },
  ]
  limit = 50
}

output "ipsec_proposals_names" {
  value = data.netbox_ipsec_proposals.filtered.items[*].name
}
