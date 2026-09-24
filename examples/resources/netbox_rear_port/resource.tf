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
resource "netbox_site" "dc1" {
  name = "DC1"
  slug = "dc1"
}
resource "netbox_device_role" "patch_panel" {
  name  = "Patch panel"
  slug  = "patch-panel"
  color = "2196f3"
}
resource "netbox_device" "pp01" {
  name           = "dc1-r01-pp01"
  device_type_id = netbox_device_type.patch_panel.id
  role_id        = netbox_device_role.patch_panel.id
  site_id        = netbox_site.dc1.id
}
resource "netbox_rear_port" "trunk1" {
  device_id = netbox_device.pp01.id
  name      = "Trunk 1"
  label     = "T1"
  type      = "mpo"
  positions = 12
  color     = "00ffff"
}
resource "netbox_rear_port" "trunk2" {
  device_id   = netbox_device.pp01.id
  name        = "Trunk 2"
  label       = "T2"
  type        = "mpo"
  positions   = 12
  description = "Spare trunk to MDF"
}
