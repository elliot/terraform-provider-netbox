resource "netbox_tunnel" "gre_branch01" {
  name          = "gre-branch01"
  status        = "active"
  encapsulation = "gre"
}

# Tunnel interfaces on the two endpoints (devices managed elsewhere).
resource "netbox_interface" "hub_tun101" {
  device_id = netbox_device.hub.id
  name      = "Tunnel101"
  type      = "virtual"
}

resource "netbox_interface" "branch_tun101" {
  device_id = netbox_device.branch01.id
  name      = "Tunnel101"
  type      = "virtual"
}

# Public address the branch uses as the tunnel source.
resource "netbox_ip_address" "branch_wan" {
  address              = "203.0.113.10/30"
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.branch_wan.id
}

# `role` is mandatory in the NetBox API (peer, hub or spoke).
resource "netbox_tunnel_termination" "hub" {
  tunnel_id        = netbox_tunnel.gre_branch01.id
  role             = "hub"
  termination_type = "dcim.interface" # or "virtualization.vminterface"
  termination_id   = netbox_interface.hub_tun101.id
}

resource "netbox_tunnel_termination" "branch" {
  tunnel_id        = netbox_tunnel.gre_branch01.id
  role             = "spoke"
  termination_type = "dcim.interface"
  termination_id   = netbox_interface.branch_tun101.id
  outside_ip_id    = netbox_ip_address.branch_wan.id
}
