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
  type           = "mpo"
  positions      = 12
}
resource "netbox_front_port_template" "lc" {
  count          = 12
  device_type_id = netbox_device_type.patch_panel.id
  name           = "LC ${count.index + 1}"
  label          = tostring(count.index + 1)
  type           = "lc"
  rear_ports = [
    {
      position           = 1
      rear_port          = netbox_rear_port_template.trunk1.id
      rear_port_position = count.index + 1
    },
  ]
}
