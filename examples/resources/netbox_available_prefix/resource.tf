# Hand-maintained example for netbox_available_prefix.

resource "netbox_prefix" "campus" {
  prefix      = "10.40.0.0/16"
  status      = "container"
  description = "Campus supernet"
}

resource "netbox_site" "hq" {
  name = "Headquarters"
  slug = "hq"
}

# Carve the next free /24 out of the campus supernet.
resource "netbox_available_prefix" "hq_users" {
  parent_prefix_id = netbox_prefix.campus.id
  prefix_length    = 24
  status           = "active"
  description      = "HQ user VLAN"
  scope_type       = "dcim.site"
  scope_id         = netbox_site.hq.id
  is_pool          = false
}

# Allocations can be chained: a /28 out of the /24 allocated above.
resource "netbox_available_prefix" "hq_printers" {
  parent_prefix_id = netbox_available_prefix.hq_users.id
  prefix_length    = 28
  description      = "HQ printers"
  mark_utilized    = true
}

output "hq_users_prefix" {
  value = netbox_available_prefix.hq_users.prefix # e.g. "10.40.0.0/24"
}
