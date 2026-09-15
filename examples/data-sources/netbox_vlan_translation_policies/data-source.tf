data "netbox_vlan_translation_policies" "all" {}

data "netbox_vlan_translation_policies" "filtered" {
  filters = [
    { name = "q", value = "customer" },
  ]
  limit = 50
}

output "vlan_translation_policies_ids" {
  value = data.netbox_vlan_translation_policies.filtered.items[*].id
}
