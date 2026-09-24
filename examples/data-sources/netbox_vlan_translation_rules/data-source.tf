data "netbox_vlan_translation_rules" "all" {}

data "netbox_vlan_translation_rules" "filtered" {
  filters = [
    { name = "policy", value = "customer-a" },
  ]
  limit = 50
}

output "vlan_translation_rules_ids" {
  value = data.netbox_vlan_translation_rules.filtered.items[*].id
}
