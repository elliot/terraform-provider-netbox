resource "netbox_tunnel_group" "branch_vpn" {
  name = "Branch VPN"
  slug = "branch-vpn"
}

# Plain GRE tunnel. `status` is mandatory in the NetBox API.
resource "netbox_tunnel" "gre_branch01" {
  name          = "gre-branch01"
  description   = "GRE overlay to branch01"
  status        = "active"
  encapsulation = "gre"
  group_id      = netbox_tunnel_group.branch_vpn.id
  tunnel_id     = 101 # local tunnel identifier, e.g. the tunnel interface number
}

# IPsec tunnel referencing an IPsec profile (see netbox_ipsec_profile).
resource "netbox_tunnel" "ipsec_branch02" {
  name             = "ipsec-branch02"
  status           = "planned"
  encapsulation    = "ipsec-tunnel"
  group_id         = netbox_tunnel_group.branch_vpn.id
  ipsec_profile_id = netbox_ipsec_profile.branch.id
  tenant_id        = netbox_tenant.acme.id
  comments         = "Waiting for the ISP hand-over."
}
