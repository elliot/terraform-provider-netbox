resource "netbox_manufacturer" "generic" {
  name = "Generic"
  slug = "generic"
}
resource "netbox_device_type" "patch_panel" {
  manufacturer_id = netbox_manufacturer.generic.id
  model           = "24-port LC patch panel"
  slug            = "pp-24-lc"
  u_height        = 1
}
resource "netbox_rear_port_template" "trunk1" {
  device_type_id = netbox_device_type.patch_panel.id
  name           = "Trunk 1"
  label          = "T1"
  type           = "mpo"
  positions      = 12
  color          = "00ffff"
}
