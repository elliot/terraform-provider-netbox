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
# A 12-strand MPO trunk on the back of the panel...
resource "netbox_rear_port" "trunk1" {
  device_id = netbox_device.pp01.id
  name      = "Trunk 1"
  label     = "T1"
  type      = "mpo"
  positions = 12
  color     = "00ffff"
}
# ...fans out to 12 LC front ports, each mapped to one trunk position.
# NetBox 4.7.0 rejects an update that re-sends an unchanged mapping set, so
# change other attributes together with the mapping or taint the port
# (see docs/validation/dcim-b.md, G1).
resource "netbox_front_port" "lc" {
  count     = 12
  device_id = netbox_device.pp01.id
  name      = "LC ${count.index + 1}"
  label     = tostring(count.index + 1)
  type      = "lc"
  rear_ports = [
    {
      position           = 1
      rear_port          = netbox_rear_port.trunk1.id
      rear_port_position = count.index + 1
    },
  ]
}
