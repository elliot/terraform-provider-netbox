resource "netbox_vlan_translation_policy" "customer_a" {
  name        = "customer-a"
  description = "Map customer A VLANs onto the provider core"
}
