resource "netbox_fhrp_group" "office_gw" {
  name        = "office-gateway"
  protocol    = "vrrp3"
  group_id    = 20
  auth_type   = "plaintext"
  auth_key    = "changeme"
  description = "Office VLAN 20 gateway"
}
