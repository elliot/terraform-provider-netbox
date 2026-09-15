# End-to-end VPN scenario for the NetBox provider.
#
# It models the crypto stack (IKE proposal -> IKE policy, IPsec proposal ->
# IPsec policy -> IPsec profile), a tunnel group with one IPsec tunnel that
# terminates on the tunnel interfaces of two devices, and a VXLAN EVPN L2VPN
# terminated on a VLAN. Everything is prefixed so it can be swept from a
# shared instance (`make sweep` removes tfacc-* objects).
#
#   export NETBOX_SERVER_URL=https://demo.netbox.dev NETBOX_API_TOKEN=...
#   terraform init && terraform apply && terraform destroy

terraform {
  required_providers {
    netbox = {
      source = "elliot/netbox"
    }
  }
}

provider "netbox" {
  requests_per_second = 2 # be polite to the shared demo instance
}

variable "prefix" {
  type        = string
  default     = "tfacc-vpn-demo"
  description = "Prefix for every object name and slug created by this scenario."
}

# --------------------------------------------------------------------------
# Phase 1 (IKE)
# --------------------------------------------------------------------------

resource "netbox_ike_proposal" "aes256_sha256" {
  name                     = "${var.prefix}-ike-aes256-sha256-dh14"
  description              = "IKEv2 proposal, CBC cipher with HMAC integrity"
  authentication_method    = "preshared-keys"
  encryption_algorithm     = "aes-256-cbc"
  authentication_algorithm = "hmac-sha256"
  group                    = 14
  sa_lifetime              = 86400
}

resource "netbox_ike_policy" "ikev2" {
  name          = "${var.prefix}-ikev2-psk"
  description   = "IKEv2 with pre-shared keys (mode is only valid for IKEv1)"
  version       = 2
  proposal_ids  = [netbox_ike_proposal.aes256_sha256.id]
  preshared_key = "${var.prefix}-change-me"
}

# --------------------------------------------------------------------------
# Phase 2 (IPsec)
# --------------------------------------------------------------------------

resource "netbox_ipsec_proposal" "esp_aes256_sha256" {
  name                     = "${var.prefix}-esp-aes256-sha256"
  encryption_algorithm     = "aes-256-cbc"
  authentication_algorithm = "hmac-sha256"
  sa_lifetime_seconds      = 3600
  sa_lifetime_data         = 4608000
}

resource "netbox_ipsec_policy" "branch" {
  name         = "${var.prefix}-ipsec-branch"
  description  = "Phase 2 policy with PFS group 14"
  proposal_ids = [netbox_ipsec_proposal.esp_aes256_sha256.id]
  pfs_group    = 14
}

resource "netbox_ipsec_profile" "branch" {
  name            = "${var.prefix}-branch-s2s"
  description     = "ESP tunnel-mode profile for branch offices"
  mode            = "esp"
  ike_policy_id   = netbox_ike_policy.ikev2.id
  ipsec_policy_id = netbox_ipsec_policy.branch.id
}

# --------------------------------------------------------------------------
# Devices carrying the tunnel end-points
# --------------------------------------------------------------------------

resource "netbox_site" "dc" {
  name = "${var.prefix}-dc1"
  slug = "${var.prefix}-dc1"
}

resource "netbox_site" "branch" {
  name = "${var.prefix}-branch01"
  slug = "${var.prefix}-branch01"
}

resource "netbox_manufacturer" "vendor" {
  name = "${var.prefix}-vendor"
  slug = "${var.prefix}-vendor"
}

resource "netbox_device_type" "edge" {
  manufacturer_id = netbox_manufacturer.vendor.id
  model           = "${var.prefix}-edge-router"
  slug            = "${var.prefix}-edge-router"
}

resource "netbox_device_role" "vpn_gateway" {
  name = "${var.prefix}-vpn-gateway"
  slug = "${var.prefix}-vpn-gateway"
}

resource "netbox_device" "hub" {
  name           = "${var.prefix}-hub-rtr1"
  site_id        = netbox_site.dc.id
  device_type_id = netbox_device_type.edge.id
  role_id        = netbox_device_role.vpn_gateway.id
}

resource "netbox_device" "spoke" {
  name           = "${var.prefix}-branch01-rtr1"
  site_id        = netbox_site.branch.id
  device_type_id = netbox_device_type.edge.id
  role_id        = netbox_device_role.vpn_gateway.id
}

resource "netbox_interface" "hub_wan" {
  device_id = netbox_device.hub.id
  name      = "GigabitEthernet0/0"
  type      = "1000base-t"
}

resource "netbox_interface" "hub_tun" {
  device_id = netbox_device.hub.id
  name      = "Tunnel101"
  type      = "virtual"
}

resource "netbox_interface" "spoke_wan" {
  device_id = netbox_device.spoke.id
  name      = "GigabitEthernet0/0"
  type      = "1000base-t"
}

resource "netbox_interface" "spoke_tun" {
  device_id = netbox_device.spoke.id
  name      = "Tunnel101"
  type      = "virtual"
}

resource "netbox_ip_address" "hub_wan" {
  address              = "198.51.100.1/30"
  description          = "${var.prefix} hub outside address"
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.hub_wan.id
}

resource "netbox_ip_address" "spoke_wan" {
  address              = "203.0.113.1/30"
  description          = "${var.prefix} branch outside address"
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.spoke_wan.id
}

# --------------------------------------------------------------------------
# Tunnel
# --------------------------------------------------------------------------

resource "netbox_tunnel_group" "branch_vpn" {
  name        = "${var.prefix}-branch-vpn"
  slug        = "${var.prefix}-branch-vpn"
  description = "Hub-and-spoke IPsec overlay"
}

resource "netbox_tunnel" "branch01" {
  name             = "${var.prefix}-ipsec-branch01"
  description      = "IPsec tunnel between dc1 and branch01"
  status           = "active"
  encapsulation    = "ipsec-tunnel"
  group_id         = netbox_tunnel_group.branch_vpn.id
  ipsec_profile_id = netbox_ipsec_profile.branch.id
  tunnel_id        = 101
}

resource "netbox_tunnel_termination" "hub" {
  tunnel_id        = netbox_tunnel.branch01.id
  role             = "hub"
  termination_type = "dcim.interface"
  termination_id   = netbox_interface.hub_tun.id
  outside_ip_id    = netbox_ip_address.hub_wan.id
}

resource "netbox_tunnel_termination" "spoke" {
  tunnel_id        = netbox_tunnel.branch01.id
  role             = "spoke"
  termination_type = "dcim.interface"
  termination_id   = netbox_interface.spoke_tun.id
  outside_ip_id    = netbox_ip_address.spoke_wan.id
}

# --------------------------------------------------------------------------
# L2VPN
# --------------------------------------------------------------------------

resource "netbox_route_target" "overlay" {
  name        = "65000:100100"
  description = "${var.prefix} overlay route target"
}

resource "netbox_l2vpn" "overlay" {
  name              = "${var.prefix}-overlay"
  slug              = "${var.prefix}-overlay"
  description       = "VXLAN EVPN overlay stretched to branch01"
  type              = "vxlan-evpn"
  identifier        = 100100
  status            = "active"
  import_target_ids = [netbox_route_target.overlay.id]
  export_target_ids = [netbox_route_target.overlay.id]
}

resource "netbox_vlan" "servers" {
  name    = "${var.prefix}-servers"
  vid     = 100
  site_id = netbox_site.dc.id
}

resource "netbox_l2vpn_termination" "servers" {
  l2vpn_id             = netbox_l2vpn.overlay.id
  assigned_object_type = "ipam.vlan"
  assigned_object_id   = netbox_vlan.servers.id
}

# --------------------------------------------------------------------------
# Read back through data sources
# --------------------------------------------------------------------------

data "netbox_tunnel_terminations" "branch01" {
  filters = [{ name = "tunnel_id", value = tostring(netbox_tunnel.branch01.id) }]
  depends_on = [netbox_tunnel_termination.hub, netbox_tunnel_termination.spoke]
}

data "netbox_ipsec_profile" "branch" {
  name       = netbox_ipsec_profile.branch.name
  depends_on = [netbox_ipsec_profile.branch]
}

output "tunnel_termination_roles" {
  value = data.netbox_tunnel_terminations.branch01.items[*].role
}

output "ipsec_profile" {
  value = {
    id     = data.netbox_ipsec_profile.branch.id
    mode   = data.netbox_ipsec_profile.branch.mode
    tunnel = netbox_tunnel.branch01.name
  }
}
