resource "netbox_tunnel_group" "branch_vpn" {
  name        = "Branch VPN"
  slug        = "branch-vpn"
  description = "Hub-and-spoke IPsec overlay towards the data centre"
}
