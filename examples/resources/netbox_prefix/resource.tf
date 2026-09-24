resource "netbox_vrf" "acme" {
  name = "ACME-L3VPN"
  rd   = "65000:100"
}

resource "netbox_site" "dc1" {
  name = "DC1"
  slug = "dc1"
}

resource "netbox_ipam_role" "production" {
  name = "Production"
  slug = "production"
}

# A container prefix ...
resource "netbox_prefix" "dc1" {
  prefix      = "10.10.0.0/16"
  vrf_id      = netbox_vrf.acme.id
  scope_type  = "dcim.site"
  scope_id    = netbox_site.dc1.id
  status      = "container"
  description = "DC1 supernet"
}

# ... and a child pool that hands out /32s.
resource "netbox_prefix" "dc1_loopbacks" {
  prefix        = "10.10.255.0/24"
  vrf_id        = netbox_vrf.acme.id
  scope_type    = "dcim.site"
  scope_id      = netbox_site.dc1.id
  role_id       = netbox_ipam_role.production.id
  status        = "active"
  is_pool       = true
  mark_utilized = false
  description   = "Loopbacks"
}
