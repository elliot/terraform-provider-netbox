resource "netbox_manufacturer" "apc" {
  name = "APC"
  slug = "apc"
}

# A rack type captures the physical properties of a rack model; racks that
# reference it inherit form factor, width, height and dimensions.
resource "netbox_rack_type" "ar3100" {
  manufacturer_id = netbox_manufacturer.apc.id
  model           = "NetShelter SX 42U"
  slug            = "apc-netshelter-sx-42u"
  form_factor     = "4-post-cabinet"
  width           = 19
  u_height        = 42
  starting_unit   = 1
  outer_width     = 600
  outer_depth     = 1070
  outer_height    = 1991
  outer_unit      = "mm"
  weight          = 122.5
  max_weight      = 1363
  weight_unit     = "kg"
  mounting_depth  = 900
  description     = "APC NetShelter SX 42U 600mm x 1070mm"
}
