resource "netbox_vlan_group" "dc1" {
  name       = "DC1 VLANs"
  slug       = "dc1-vlans"
  vid_ranges = [[100, 199]]
}

resource "netbox_ipam_role" "production" {
  name = "Production"
  slug = "production"
}

resource "netbox_vlan" "servers" {
  group_id    = netbox_vlan_group.dc1.id
  vid         = 110
  name        = "SERVERS"
  status      = "active"
  role_id     = netbox_ipam_role.production.id
  description = "Server access VLAN"
}
