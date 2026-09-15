resource "netbox_vrf" "acme" {
  name = "ACME-L3VPN"
}

resource "netbox_ipam_role" "dhcp" {
  name = "DHCP"
  slug = "dhcp"
}

# Both ends carry the prefix length of the enclosing network.
resource "netbox_ip_range" "dhcp_pool" {
  start_address = "10.10.20.100/24"
  end_address   = "10.10.20.199/24"
  vrf_id        = netbox_vrf.acme.id
  role_id       = netbox_ipam_role.dhcp.id
  status        = "active"
  mark_utilized = true
  description   = "DHCP pool for office VLAN 20"
}
