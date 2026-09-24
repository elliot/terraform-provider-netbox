resource "netbox_l2vpn" "tenant_a" {
  name       = "tenant-a-overlay"
  slug       = "tenant-a-overlay"
  type       = "vxlan-evpn"
  identifier = 100100
}

resource "netbox_vlan" "tenant_a_servers" {
  name = "tenant-a-servers"
  vid  = 100
}

# Attach a VLAN to the L2VPN ...
resource "netbox_l2vpn_termination" "vlan" {
  l2vpn_id             = netbox_l2vpn.tenant_a.id
  assigned_object_type = "ipam.vlan"
  assigned_object_id   = netbox_vlan.tenant_a_servers.id
}

# ... or a device interface (also virtualization.vminterface).
resource "netbox_l2vpn_termination" "interface" {
  l2vpn_id             = netbox_l2vpn.tenant_a.id
  assigned_object_type = "dcim.interface"
  assigned_object_id   = netbox_interface.leaf1_eth1.id
}
