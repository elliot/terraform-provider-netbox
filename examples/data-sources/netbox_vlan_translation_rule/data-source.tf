data "netbox_vlan_translation_rule" "example" {
  id = 123
}

# Any API filter of /api/ipam/vlan-translation-rules/ works; the lookup must match exactly one object.
data "netbox_vlan_translation_rule" "filtered" {
  filters = [
    { name = "policy", value = "customer-a" },
    { name = "local_vid", value = "100" },
  ]
}

output "vlan_translation_rule_id" {
  value = data.netbox_vlan_translation_rule.example.id
}
