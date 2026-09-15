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
resource "netbox_site" "dc1" {
  name = "DC1"
  slug = "dc1"
}
resource "netbox_device_role" "console_server" {
  name  = "Console server"
  slug  = "console-server"
  color = "2196f3"
}
resource "netbox_device" "con01" {
  name           = "dc1-con01"
  device_type_id = netbox_device_type.om2248.id
  role_id        = netbox_device_role.console_server.id
  site_id        = netbox_site.dc1.id
}
resource "netbox_console_server_port" "ports" {
  count       = 48
  device_id   = netbox_device.con01.id
  name        = "port${count.index + 1}"
  label       = tostring(count.index + 1)
  type        = "rj-45"
  speed       = 9600
  description = "Serial line ${count.index + 1}"
}
