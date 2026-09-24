resource "netbox_site" "fra1" {
  name = "FRA1"
  slug = "fra1"
}

resource "netbox_rack" "a01" {
  name    = "A01"
  site_id = netbox_site.fra1.id
}

resource "netbox_user" "jdoe" {
  username = "jdoe"
  password = "change-me-please"
}

resource "netbox_tenant" "acme" {
  name = "ACME Corp"
  slug = "acme"
}

# Reserve rack units 1-4 for an upcoming installation. `units` is a set of
# unit numbers; they must exist in the rack and not overlap other reservations.
resource "netbox_rack_reservation" "a01_storage" {
  rack_id     = netbox_rack.a01.id
  user_id     = netbox_user.jdoe.id
  tenant_id   = netbox_tenant.acme.id
  units       = [1, 2, 3, 4]
  status      = "active"
  description = "Reserved for storage array delivery in Q3"
}
