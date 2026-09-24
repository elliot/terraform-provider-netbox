resource "netbox_fhrp_group" "office_gw" {
  name        = "office-gateway"
  protocol    = "vrrp3"
  group_id    = 20
  auth_type   = "plaintext"
  auth_key    = var.vrrp_auth_key
  description = "Office VLAN 20 gateway"
}

variable "vrrp_auth_key" {
  type      = string
  sensitive = true
}
