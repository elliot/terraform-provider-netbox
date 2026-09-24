resource "netbox_site" "dc1" {
  name = "DC1"
  slug = "dc1"
}

# vid_ranges is a list of [min, max] pairs.
resource "netbox_vlan_group" "dc1" {
  name        = "DC1 VLANs"
  slug        = "dc1-vlans"
  scope_type  = "dcim.site"
  scope_id    = netbox_site.dc1.id
  vid_ranges  = [[100, 199], [1000, 1999]]
  description = "VLAN IDs available in DC1"
}
