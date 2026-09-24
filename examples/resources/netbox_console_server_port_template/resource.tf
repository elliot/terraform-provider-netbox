resource "netbox_manufacturer" "opengear" {
  name = "Opengear"
  slug = "opengear"
}
resource "netbox_device_type" "om2248" {
  manufacturer_id = netbox_manufacturer.opengear.id
  model           = "OM2248"
  slug            = "om2248"
  u_height        = 1
}
resource "netbox_console_server_port_template" "ports" {
  count          = 48
  device_type_id = netbox_device_type.om2248.id
  name           = "port${count.index + 1}"
  label          = tostring(count.index + 1)
  type           = "rj-45"
}
