resource "netbox_vlan_translation_policy" "customer_a" {
  name = "customer-a"
}

resource "netbox_vlan_translation_rule" "customer_a_100" {
  policy_id   = netbox_vlan_translation_policy.customer_a.id
  local_vid   = 100
  remote_vid  = 2100
  description = "Customer VLAN 100 -> core 2100"
}
