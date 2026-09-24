resource "netbox_site" "fra1" {
  name = "FRA1"
  slug = "fra1"
}

resource "netbox_tenant" "acme" {
  name = "ACME Corp"
  slug = "acme"
}

resource "netbox_location" "floor_2" {
  name        = "Floor 2"
  slug        = "fra1-floor-2"
  site_id     = netbox_site.fra1.id
  status      = "active"
  description = "Second floor data hall"
}

# Locations nest within a site; the parent must belong to the same site.
resource "netbox_location" "cage_a" {
  name        = "Cage A"
  slug        = "fra1-floor-2-cage-a"
  site_id     = netbox_site.fra1.id
  parent_id   = netbox_location.floor_2.id
  tenant_id   = netbox_tenant.acme.id
  status      = "active"
  facility    = "Cage 2A"
  description = "ACME private cage"
}
