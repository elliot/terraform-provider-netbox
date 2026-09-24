data "netbox_vlan_translation_policy" "example" {
  name = "customer-a"
}

# Any API filter of /api/ipam/vlan-translation-policies/ works; the lookup must match exactly one object.
data "netbox_vlan_translation_policy" "filtered" {
  filters = [
    { name = "id", value = "123" },
  ]
}

output "vlan_translation_policy_id" {
  value = data.netbox_vlan_translation_policy.example.id
}
